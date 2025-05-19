package app

import (
	"context"
	"encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/lock"
	"gofire/internal/models"
	"gofire/internal/repository"
	"gofire/internal/state"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
	"time"
)

type JobResult struct {
	JobID       int64
	Err         error
	Attempts    int
	MaxAttempts int
	Status      state.JobStatus
}

type EnqueueScheduler struct {
	repository repository.EnqueuedJobRepository
	instance   string
	lock       lock.DistributedLockManager
	handlers   map[string]func([]interface{}) error
	jobResults chan JobResult
}

func NewScheduler(repository repository.EnqueuedJobRepository, lock lock.DistributedLockManager, instance string) EnqueueScheduler {
	scheduler := EnqueueScheduler{
		repository: repository,
		instance:   instance,
		lock:       lock,
		handlers:   make(map[string]func([]interface{}) error),
		jobResults: make(chan JobResult, 1000),
	}

	go scheduler.MarkRetryFailedJobs(context.Background())
	return scheduler
}

func (s *EnqueueScheduler) RegisterHandler(name string, fn func([]interface{}) error) {
	s.handlers[name] = fn
}

func (s *EnqueueScheduler) GetHandler(name string) (func([]interface{}) error, error) {
	fn, ok := s.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return fn, nil
}

func (s *EnqueueScheduler) Enqueue(ctx context.Context, name string, scheduledAt time.Time, args []interface{}) (int64, error) {
	if jobID, err := s.repository.Insert(ctx, name, scheduledAt, args); err != nil {
		log.Println(err.Error())
		return 0, err
	} else {
		return jobID, nil
	}
}

func (s *EnqueueScheduler) MarkRetryFailedJobs(ctx context.Context) {

	const retryLock = constants.RetryLock
	if err := s.lock.Acquire(retryLock); err != nil {
		return
	}
	defer s.lock.Release(retryLock)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.repository.MarkRetryFailedJobs(ctx)
		}
	}
}

func (s *EnqueueScheduler) ProcessEnqueues(ctx context.Context, interval, workerCount int) error {
	const enqueueLock = constants.EnqueueLock
	if err := s.lock.Acquire(enqueueLock); err != nil {
		return err
	}
	defer s.lock.Release(enqueueLock)

	s.startResultProcessor(ctx)

	if err := s.repository.UnlockStaleJobs(ctx, 5*time.Minute); err != nil {
		log.Fatal(err.Error())
	}

	sem := semaphore.NewWeighted(int64(workerCount))
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
			s.processDueJobs(ctx, sem, &wg)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}
}

func (s *EnqueueScheduler) ExecuteJobManually(ctx context.Context, jobID int64) error {
	job, err := s.repository.FindByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	handler, err := s.GetHandler(job.Name)
	if err != nil {
		return fmt.Errorf("handler not found: %w", err)
	}

	var args []interface{}
	if err := json.Unmarshal(job.Payload, &args); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	err = handler(args)
	status := s.errorToJobStatus(err)

	s.jobResults <- JobResult{
		JobID:       job.ID,
		Err:         err,
		Attempts:    job.Attempts + 1,
		MaxAttempts: job.MaxAttempts,
		Status:      status,
	}

	return nil
}

func (s *EnqueueScheduler) startResultProcessor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-s.jobResults:
				switch res.Status {
				case state.StatusSucceeded:
					if state.IsValidTransition(state.StatusProcessing, state.StatusSucceeded) {
						s.repository.MarkSuccess(ctx, res.JobID)
					}
				case state.StatusFailed:
					if state.IsValidTransition(state.StatusProcessing, state.StatusFailed) {
						s.repository.MarkFailure(ctx, res.JobID, res.Err.Error(), res.Attempts, res.MaxAttempts)
					}
				default:
					log.Printf("unknown job status: %s", res.Status)
				}
			}
		}
	}()
}

func (s *EnqueueScheduler) processDueJobs(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup) {
	log.Println("start to process enqueued jobs")
	now := time.Now()
	statuses := []state.JobStatus{state.StatusQueued, state.StatusRetrying}
	jobsList, err := s.repository.FetchDueJobs(ctx, 1, 100, statuses, &now)
	if err != nil {
		log.Println("fetch error:", err)
		time.Sleep(2 * time.Second)
		return
	}

	for _, job := range jobsList.Items {
		ok, err := s.repository.LockJob(ctx, &job, s.instance)
		if err != nil || !ok {
			log.Println(err)
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			log.Println(err.Error())
			break
		}
		wg.Add(1)

		go s.handleJob(ctx, sem, wg, job)
	}
}

func (s *EnqueueScheduler) handleJob(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, job models.EnqueuedJob) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in job %d: %v", job.ID, r)
		}
		sem.Release(1)
		wg.Done()
	}()

	handler, err := s.GetHandler(job.Name)
	if err != nil {
		log.Println(err.Error())
		s.jobResults <- JobResult{
			JobID:       job.ID,
			Err:         fmt.Errorf("handler not found"),
			Attempts:    job.Attempts + 1,
			MaxAttempts: job.MaxAttempts,
			Status:      state.StatusFailed,
		}
		return
	}

	var args []interface{}
	if err := json.Unmarshal(job.Payload, &args); err != nil {
		log.Printf("invalid payload for job %d: %v", job.ID, err)
		s.jobResults <- JobResult{
			JobID:       job.ID,
			Err:         fmt.Errorf("invalid payload"),
			Attempts:    job.Attempts + 1,
			MaxAttempts: job.MaxAttempts,
			Status:      state.StatusFailed,
		}
		return
	}

	err = handler(args)

	status := s.errorToJobStatus(err)
	s.jobResults <- JobResult{
		JobID:       job.ID,
		Err:         err,
		Attempts:    job.Attempts + 1,
		MaxAttempts: job.MaxAttempts,
		Status:      status,
	}
}

func (*EnqueueScheduler) errorToJobStatus(err error) state.JobStatus {
	if err != nil {
		return state.StatusFailed
	}
	return state.StatusSucceeded
}
