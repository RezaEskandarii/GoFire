package gofire

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

type enqueueJobsManager struct {
	repository repository.EnqueuedJobRepository
	instance   string
	lock       lock.DistributedLockManager
	jobHandler JobHandler
	jobResults chan models.JobResult
}

func newEnqueueScheduler(repository repository.EnqueuedJobRepository, lock lock.DistributedLockManager, jobHandler JobHandler, instance string) enqueueJobsManager {
	scheduler := enqueueJobsManager{
		repository: repository,
		instance:   instance,
		lock:       lock,
		jobHandler: jobHandler,
		jobResults: make(chan models.JobResult, 1000),
	}
	go scheduler.MarkRetryFailedJobs(context.Background())
	return scheduler
}

func (em *enqueueJobsManager) Enqueue(ctx context.Context, name string, scheduledAt time.Time, args []interface{}) (int64, error) {
	if jobID, err := em.repository.Insert(ctx, name, scheduledAt, args); err != nil {
		log.Println(err.Error())
		return 0, err
	} else {
		return jobID, nil
	}
}

func (em *enqueueJobsManager) MarkRetryFailedJobs(ctx context.Context) {

	const retryLock = constants.RetryLock
	if err := em.lock.Acquire(retryLock); err != nil {
		return
	}
	defer em.lock.Release(retryLock)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			em.repository.MarkRetryFailedJobs(ctx)
		}
	}
}

func (em *enqueueJobsManager) Start(ctx context.Context, interval, workerCount, batchSize int) error {
	const enqueueLock = constants.StartEnqueueLock
	if err := em.lock.Acquire(enqueueLock); err != nil {
		return err
	}
	defer em.lock.Release(enqueueLock)

	em.startResultProcessor(ctx)

	if err := em.repository.UnlockStaleJobs(ctx, 5*time.Minute); err != nil {
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
			em.processDueJobs(ctx, sem, &wg, batchSize)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}
}

func (em *enqueueJobsManager) ExecuteJobManually(ctx context.Context, jobID int64) error {
	job, err := em.repository.FindByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	ok := em.jobHandler.Exists(job.Name)
	if !ok {
		return fmt.Errorf("handler not found: %w", err)
	}

	var args []interface{}
	if err := json.Unmarshal(job.Payload, &args); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	err = em.jobHandler.Execute(job.Name, args)
	status := em.errorToJobStatus(err)

	em.jobResults <- models.JobResult{
		JobID:       job.ID,
		Err:         err,
		Attempts:    job.Attempts + 1,
		MaxAttempts: job.MaxAttempts,
		Status:      status,
	}

	return nil
}

func (em *enqueueJobsManager) startResultProcessor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-em.jobResults:
				switch res.Status {
				case state.StatusSucceeded:
					if state.IsValidTransition(state.StatusProcessing, state.StatusSucceeded) {
						em.repository.MarkSuccess(ctx, res.JobID)
					}
				case state.StatusFailed:
					if state.IsValidTransition(state.StatusProcessing, state.StatusFailed) {
						em.repository.MarkFailure(ctx, res.JobID, res.Err.Error(), res.Attempts, res.MaxAttempts)
					}
				default:
					log.Printf("unknown job status: %em", res.Status)
				}
			}
		}
	}()
}

func (em *enqueueJobsManager) processDueJobs(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, batchSize int) {
	log.Println("start to process enqueued jobs")
	now := time.Now()
	statuses := []state.JobStatus{state.StatusQueued, state.StatusRetrying}
	jobsList, err := em.repository.FetchDueJobs(ctx, 1, batchSize, statuses, &now)
	if err != nil {
		log.Println("fetch error:", err)
		time.Sleep(2 * time.Second)
		return
	}

	for _, job := range jobsList.Items {
		ok, err := em.repository.LockJob(ctx, job.ID, em.instance)
		if err != nil || !ok {
			log.Println(err)
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			log.Println(err.Error())
			break
		}
		wg.Add(1)

		go em.handleJob(ctx, sem, wg, job)
	}
}

func (em *enqueueJobsManager) handleJob(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, job models.EnqueuedJob) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in job %d: %v", job.ID, r)
		}
		sem.Release(1)
		wg.Done()
	}()

	ok := em.jobHandler.Exists(job.Name)
	if !ok {
		em.jobResults <- models.JobResult{
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
		em.jobResults <- models.JobResult{
			JobID:       job.ID,
			Err:         fmt.Errorf("invalid payload"),
			Attempts:    job.Attempts + 1,
			MaxAttempts: job.MaxAttempts,
			Status:      state.StatusFailed,
		}
		return
	}

	err := em.jobHandler.Execute(job.Name, args)

	status := em.errorToJobStatus(err)
	em.jobResults <- models.JobResult{
		JobID:       job.ID,
		Err:         err,
		Attempts:    job.Attempts + 1,
		MaxAttempts: job.MaxAttempts,
		Status:      status,
	}
}

func (*enqueueJobsManager) errorToJobStatus(err error) state.JobStatus {
	if err != nil {
		return state.StatusFailed
	}
	return state.StatusSucceeded
}
