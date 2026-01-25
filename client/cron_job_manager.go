package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/pgk/parser"
	"github.com/RezaEskandarii/gofire/types"
	"github.com/RezaEskandarii/gofire/types/config"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type CronJobManager struct {
	jobStore   store.CronJobStore
	lock       lock.DistributedLockManager
	jobHandler *config.JobHandler
	instance   string
	jobResults chan types.JobResult
}

func NewCronJobManager(cronJobStore store.CronJobStore, lock lock.DistributedLockManager, jobHandler *config.JobHandler, instance string) CronJobManager {
	scheduler := CronJobManager{
		jobStore:   cronJobStore,
		lock:       lock,
		jobHandler: jobHandler,
		instance:   instance,
		jobResults: make(chan types.JobResult, 1000),
	}
	go scheduler.startResultProcessor(context.Background())
	return scheduler
}

func (cm *CronJobManager) Start(ctx context.Context, intervalSeconds, workerCount, batchSize int) error {

	sem := semaphore.NewWeighted(int64(workerCount))
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			log.Println("CronJobManager stopped")
			wg.Wait()
			return ctx.Err()
		default:
			cm.processCronJobs(ctx, sem, &wg, batchSize)
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}
}

func (cm *CronJobManager) processCronJobs(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, batchSize int) {

	log.Println("start to process cron jobs")
	page := 1
	for {
		result, err := cm.jobStore.FetchDueCronJobs(ctx, page, batchSize)
		if err != nil {
			log.Printf("CronJobManager: failed to fetch jobs: %v", err)
			return
		}

		for _, job := range result.Items {
			ok, err := cm.jobStore.LockJob(ctx, job.ID, cm.instance)
			if err != nil || !ok {
				log.Println(err)
				continue
			}
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Println("CronJobManager: semaphore error:", err)
				continue
			}
			wg.Add(1)

			go func(job types.CronJob) {
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

func (cm *CronJobManager) executeJob(ctx context.Context, job types.CronJob) {
	now := time.Now()
	nextRun := parser.CalculateNextRun(job.Expression, now)

	if !cm.jobHandler.Exists(job.Name) {
		cm.jobResults <- types.JobResult{
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
		cm.jobResults <- types.JobResult{
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

	cm.jobResults <- types.JobResult{
		JobID:   job.ID,
		Err:     err,
		Status:  status,
		RanAt:   now,
		NextRun: nextRun,
	}
}

func (cm *CronJobManager) startResultProcessor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-cm.jobResults:
				switch res.Status {
				case state.StatusSucceeded:
					if state.IsValidTransition(state.StatusProcessing, state.StatusSucceeded) {
						if err := cm.jobStore.MarkSuccess(ctx, res.JobID); err != nil {
							log.Printf("MarkSuccess error: %s", err.Error())
						}
					}
				case state.StatusFailed:
					if state.IsValidTransition(state.StatusProcessing, state.StatusFailed) {
						if err := cm.jobStore.MarkFailure(ctx, res.JobID, res.Err.Error()); err != nil {
							log.Printf("MarkFailure error: %s", err.Error())
						}
					}
				default:
					log.Printf("CronJobManager: unknown status: %sm", res.Status)
				}
				if err := cm.jobStore.UpdateJobRunTimes(ctx, res.JobID, res.RanAt, res.NextRun); err != nil {
					log.Printf("UpdateJobRunTimes error: %s", err.Error())
				}
			}
		}
	}()
}

func (cm *CronJobManager) errorToJobStatus(err error) state.JobStatus {
	if err != nil {
		return state.StatusFailed
	}
	return state.StatusSucceeded
}
