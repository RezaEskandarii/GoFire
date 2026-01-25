package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/RezaEskandarii/gofire/constants"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/message_broaker"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/types"
	"github.com/RezaEskandarii/gofire/types/config"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
	"time"
)

type EnqueueJobsManager struct {
	store      store.EnqueuedJobStore
	instance   string
	lock       lock.DistributedLockManager
	jobHandler *config.JobHandler
	jobResults chan types.JobResult
	mBroker    message_broaker.MessageBroker
}

func NewEnqueueScheduler(jobStore store.EnqueuedJobStore, lock lock.DistributedLockManager, jobHandler *config.JobHandler, messageBroker message_broaker.MessageBroker, instance string) EnqueueJobsManager {
	scheduler := EnqueueJobsManager{
		store:      jobStore,
		instance:   instance,
		lock:       lock,
		jobHandler: jobHandler,
		jobResults: make(chan types.JobResult, 1000),
		mBroker:    messageBroker,
	}
	go scheduler.MarkRetryFailedJobs(context.Background())
	return scheduler
}

func (em *EnqueueJobsManager) Enqueue(ctx context.Context, name string, scheduledAt time.Time, args ...any) (int64, error) {
	if jobID, err := em.store.Insert(ctx, name, scheduledAt, args...); err != nil {
		log.Println(err.Error())
		return 0, err
	} else {
		return jobID, nil
	}
}

func (em *EnqueueJobsManager) MarkRetryFailedJobs(ctx context.Context) {
	const retryLock = constants.RetryLock

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := em.lock.Acquire(retryLock); err != nil {
				return
			}
			if err := em.store.MarkRetryFailedJobs(ctx); err != nil {
				log.Printf("MarkRetryFailedJobs error: %s", err.Error())
			}
			if err := em.lock.Release(retryLock); err != nil {
				log.Printf("Release Retry Job error: %s", err.Error())
			}
		}
	}
}

func (em *EnqueueJobsManager) Start(ctx context.Context, interval, workerCount, batchSize int) error {

	em.startResultProcessor(ctx)

	if err := em.store.UnlockStaleJobs(ctx, 5*time.Minute); err != nil {
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

func (em *EnqueueJobsManager) ExecuteJobManually(ctx context.Context, jobID int64) error {
	job, err := em.store.FindByID(ctx, jobID)
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

	err = em.jobHandler.Execute(job.Name, args...)
	status := em.errorToJobStatus(err)

	em.jobResults <- types.JobResult{
		JobID:       job.ID,
		Err:         err,
		Attempts:    job.Attempts + 1,
		MaxAttempts: job.MaxAttempts,
		Status:      status,
	}

	return nil
}

func (em *EnqueueJobsManager) startResultProcessor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-em.jobResults:
				switch res.Status {
				case state.StatusSucceeded:
					if state.IsValidTransition(state.StatusProcessing, state.StatusSucceeded) {
						if err := em.store.MarkSuccess(ctx, res.JobID); err != nil {
							log.Printf("MarkSuccess error: %s", err.Error())
						}
					}
				case state.StatusFailed:
					if state.IsValidTransition(state.StatusProcessing, state.StatusFailed) {
						if err := em.store.MarkFailure(ctx, res.JobID, res.Err.Error(), res.Attempts, res.MaxAttempts); err != nil {
							log.Printf("MarkFailure error: %s", err.Error())
						}
					}
				default:
					log.Printf("unknown job status: %sm", res.Status)
				}
			}
		}
	}()
}

func (em *EnqueueJobsManager) processDueJobs(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, batchSize int) {
	log.Println("start to process enqueued jobs")
	now := time.Now()
	statuses := []state.JobStatus{state.StatusQueued, state.StatusRetrying}
	jobsList, err := em.store.FetchDueJobs(ctx, 1, batchSize, statuses, &now)
	if err != nil {
		log.Println("fetch error:", err)
		time.Sleep(2 * time.Second)
		return
	}

	for _, job := range jobsList.Items {
		ok, err := em.store.LockJob(ctx, job.ID, em.instance)
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

func (em *EnqueueJobsManager) handleJob(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, job types.EnqueuedJob) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in job %d: %v", job.ID, r)
		}
		sem.Release(1)
		wg.Done()
	}()

	ok := em.jobHandler.Exists(job.Name)
	if !ok {
		em.jobResults <- types.JobResult{
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
		em.jobResults <- types.JobResult{
			JobID:       job.ID,
			Err:         fmt.Errorf("invalid payload"),
			Attempts:    job.Attempts + 1,
			MaxAttempts: job.MaxAttempts,
			Status:      state.StatusFailed,
		}
		return
	}

	err := em.jobHandler.Execute(job.Name, args...)

	status := em.errorToJobStatus(err)
	em.jobResults <- types.JobResult{
		JobID:       job.ID,
		Err:         err,
		Attempts:    job.Attempts + 1,
		MaxAttempts: job.MaxAttempts,
		Status:      status,
	}
}

func (*EnqueueJobsManager) errorToJobStatus(err error) state.JobStatus {
	if err != nil {
		return state.StatusFailed
	}
	return state.StatusSucceeded
}

func (em *EnqueueJobsManager) StartQueueAndStorageSyncWorker(ctx context.Context, queue string, useQueue bool) error {
	if !useQueue {
		return nil
	}
	const batchSize = 1000
	log.Println("start to sync rabbit mq published jobs with database")

	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		msgCh, err := em.mBroker.Consume(ctx, queue)
		if err != nil {
			log.Printf("failed to start consuming messages: %v", err)
			return
		}

		var jobsBatch []types.Job

		flushBatch := func() {
			if len(jobsBatch) == 0 {
				return
			}
			if err := em.store.BulkInsert(ctx, jobsBatch); err != nil {
				log.Printf("failed to insert batch jobs: %v", err)
			} else {
				log.Printf("inserted %d jobs in batch", len(jobsBatch))
			}
			jobsBatch = nil
		}

		for {
			select {
			case <-ctx.Done():
				log.Println("batch job sync stopped due to context cancellation")
				flushBatch()
				return

			case msg, ok := <-msgCh:
				if !ok {
					log.Println("message channel closed")
					flushBatch()
					return
				}

				var job types.Job
				if err := json.Unmarshal(msg, &job); err != nil {
					log.Printf("failed to unmarshal job: %v", err)
					continue
				}

				jobsBatch = append(jobsBatch, job)
				if len(jobsBatch) >= batchSize {
					flushBatch()
				}

			case <-ticker.C:
				flushBatch()
			}
		}
	}()

	return nil
}
