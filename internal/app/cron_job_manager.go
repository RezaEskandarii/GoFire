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
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"golang.org/x/sync/semaphore"
)

type CronJobManager struct {
	repository repository.CronJobRepository
	lock       lock.DistributedLockManager
	jobHandler JobHandler
	instance   string
	jobResults chan models.JobResult
}

func NewCronJobManager(repo repository.CronJobRepository, lock lock.DistributedLockManager, jobHandler JobHandler, instance string) *CronJobManager {
	scheduler := &CronJobManager{
		repository: repo,
		lock:       lock,
		jobHandler: jobHandler,
		instance:   instance,
		jobResults: make(chan models.JobResult, 1000),
	}
	go scheduler.startResultProcessor(context.Background())
	return scheduler
}

func (s *CronJobManager) Start(ctx context.Context, intervalSeconds, workerCount, batchSize int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	sem := semaphore.NewWeighted(int64(workerCount))
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			log.Println("CronJobManager stopped")
			wg.Wait()
			return
		case <-ticker.C:
			s.processCronJobs(ctx, sem, &wg, batchSize)
		}
	}
}

func (s *CronJobManager) processCronJobs(ctx context.Context, sem *semaphore.Weighted, wg *sync.WaitGroup, batchSize int) {
	const cronLock = constants.CronJobLock
	if err := s.lock.Acquire(cronLock); err != nil {
		log.Println("CronJobManager: lock acquire failed:", err)
		return
	}
	defer s.lock.Release(cronLock)

	page := 1
	for {
		result, err := s.repository.FetchDueCronJobs(ctx, page, batchSize)
		if err != nil {
			log.Printf("CronJobManager: failed to fetch jobs: %v", err)
			return
		}

		for _, job := range result.Items {
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Println("CronJobManager: semaphore error:", err)
				continue
			}
			wg.Add(1)

			go func(job models.CronJob) {
				defer sem.Release(1)
				defer wg.Done()
				s.executeJob(ctx, job)
			}(job)
		}

		if !result.HasNextPage {
			break
		}
		page++
	}
}

func (s *CronJobManager) executeJob(ctx context.Context, job models.CronJob) {
	now := time.Now()
	nextRun := calculateNextRun(job.Expression, now)

	if !s.jobHandler.Exists(job.Name) {
		s.jobResults <- models.JobResult{
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
		s.jobResults <- models.JobResult{
			JobID:   job.ID,
			Err:     fmt.Errorf("invalid payload: %v", err),
			Status:  state.StatusFailed,
			RanAt:   now,
			NextRun: nextRun,
		}
		return
	}

	err := s.jobHandler.Execute(job.Name, args)
	status := s.errorToJobStatus(err)

	s.jobResults <- models.JobResult{
		JobID:   job.ID,
		Err:     err,
		Status:  status,
		RanAt:   now,
		NextRun: nextRun,
	}
}

func (s *CronJobManager) startResultProcessor(ctx context.Context) {
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
						s.repository.MarkFailure(ctx, res.JobID, res.Err.Error())
					}
				default:
					log.Printf("CronJobManager: unknown status: %s", res.Status)
				}
				s.repository.UpdateJobRunTimes(ctx, res.JobID, res.RanAt, res.NextRun)
			}
		}
	}()
}

func (s *CronJobManager) errorToJobStatus(err error) state.JobStatus {
	if err != nil {
		return state.StatusFailed
	}
	return state.StatusSucceeded
}

func calculateNextRun(expr string, from time.Time) time.Time {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expr)
	if err != nil {
		log.Printf("CronJobManager: invalid cron expression '%s': %v", expr, err)
		return from.Add(1 * time.Hour)
	}
	return schedule.Next(from)
}
