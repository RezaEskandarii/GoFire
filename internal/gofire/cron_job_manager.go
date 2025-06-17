package gofire

import (
	"context"
	"encoding/json"
	"fmt"
	"gofire/internal/lock"
	"gofire/internal/models"
	"gofire/internal/parser"
	"gofire/internal/state"
	"gofire/internal/store"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type cronJobManager struct {
	jobStore   store.CronJobStore
	lock       lock.DistributedLockManager
	jobHandler *JobHandler
	instance   string
	jobResults chan models.JobResult
}

func newCronJobManager(cronJobStore store.CronJobStore, lock lock.DistributedLockManager, jobHandler *JobHandler, instance string) cronJobManager {
	scheduler := cronJobManager{
		jobStore:   cronJobStore,
		lock:       lock,
		jobHandler: jobHandler,
		instance:   instance,
		jobResults: make(chan models.JobResult, 1000),
	}
	go scheduler.startResultProcessor(context.Background())
	return scheduler
}

func (cm *cronJobManager) Start(ctx context.Context, intervalSeconds, workerCount, batchSize int) error {

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	sem := semaphore.NewWeighted(int64(workerCount))
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			log.Println("cronJobManager stopped")
			wg.Wait()
			return ctx.Err()
		case <-ticker.C:
			cm.processCronJobs(ctx, sem, &wg, batchSize)
		}
	}
}

func (cm *cronJobManager) processCronJobs(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, batchSize int) {

	log.Println("start to process cron jobs")
	page := 1
	for {
		result, err := cm.jobStore.FetchDueCronJobs(ctx, page, batchSize)
		if err != nil {
			log.Printf("cronJobManager: failed to fetch jobs: %v", err)
			return
		}

		for _, job := range result.Items {
			ok, err := cm.jobStore.LockJob(ctx, job.ID, cm.instance)
			if err != nil || !ok {
				log.Println(err)
				continue
			}
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Println("cronJobManager: semaphore error:", err)
				continue
			}
			wg.Add(1)

			go func(job models.CronJob) {
				defer sem.Release(1)
				defer wg.Done()
				cm.executeJob(ctx, job)
			}(job)
		}

		if !result.HasNextPage {
			break
		}
		page++
	}
}

func (cm *cronJobManager) executeJob(ctx context.Context, job models.CronJob) {
	now := time.Now()
	nextRun := parser.CalculateNextRun(job.Expression, now)

	if !cm.jobHandler.Exists(job.Name) {
		cm.jobResults <- models.JobResult{
			JobID:   job.ID,
			Err:     fmt.Errorf("handler not found"),
			Status:  state.StatusFailed,
			RanAt:   now,
			NextRun: nextRun,
		}
		return
	}

	var args []interface{}
	if err := json.Unmarshal(job.Payload, &args); err != nil {
		cm.jobResults <- models.JobResult{
			JobID:   job.ID,
			Err:     fmt.Errorf("invalid payload: %v", err),
			Status:  state.StatusFailed,
			RanAt:   now,
			NextRun: nextRun,
		}
		return
	}

	err := cm.jobHandler.Execute(job.Name, args...)
	status := cm.errorToJobStatus(err)

	cm.jobResults <- models.JobResult{
		JobID:   job.ID,
		Err:     err,
		Status:  status,
		RanAt:   now,
		NextRun: nextRun,
	}
}

func (cm *cronJobManager) startResultProcessor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-cm.jobResults:
				switch res.Status {
				case state.StatusSucceeded:
					if state.IsValidTransition(state.StatusProcessing, state.StatusSucceeded) {
						cm.jobStore.MarkSuccess(ctx, res.JobID)
						cm.jobStore.UnLockJob(ctx, res.JobID)
					}
				case state.StatusFailed:
					if state.IsValidTransition(state.StatusProcessing, state.StatusFailed) {
						cm.jobStore.MarkFailure(ctx, res.JobID, res.Err.Error())
					}
				default:
					log.Printf("cronJobManager: unknown status: %sm", res.Status)
				}
				cm.jobStore.UpdateJobRunTimes(ctx, res.JobID, res.RanAt, res.NextRun)
			}
		}
	}()
}

func (cm *cronJobManager) errorToJobStatus(err error) state.JobStatus {
	if err != nil {
		return state.StatusFailed
	}
	return state.StatusSucceeded
}
