package gofire

import (
	"context"
	json2 "encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/lock"
	"gofire/internal/message_broaker"
	"gofire/internal/models"
	"gofire/internal/parser"
	"gofire/internal/store"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// JobManager defines methods for scheduling and managing background tasks.
type JobManager interface {

	// Enqueue either directly stores the job in the database (if disabled queue mode),
	// or publishes it to a message broker like RabbitMQ if enabled.
	// In queue mode, it returns 0 as job ID is unknown at this stage.
	Enqueue(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error)

	// RemoveEnqueue deletes a queued job using its ID.
	RemoveEnqueue(ctx context.Context, jobID int64) error

	// FindEnqueue returns the details of a queued job by its ID.
	FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error)

	// Schedule sets up a recurring job based on a cron expression.
	Schedule(ctx context.Context, jobName string, expression string, args ...any) (int64, error)

	// ActivateSchedule enables a scheduled job if it was previously disabled.
	ActivateSchedule(ctx context.Context, jobID int64)

	// DeActivateSchedule temporarily disables a scheduled job.
	DeActivateSchedule(ctx context.Context, jobID int64)

	// ScheduleEveryMinute runs a job once every minute.
	ScheduleEveryMinute(ctx context.Context, jobName string, args ...any) (int64, error)

	// ScheduleEveryHour runs a job once every hour.
	ScheduleEveryHour(ctx context.Context, jobName string, args ...any) (int64, error)

	// ScheduleEveryDay runs a job once every day.
	ScheduleEveryDay(ctx context.Context, jobName string, args ...any) (int64, error)

	// ScheduleEveryWeek runs a job once a week.
	ScheduleEveryWeek(ctx context.Context, jobName string, args ...any) (int64, error)

	// ScheduleEveryMonth runs a job once a month.
	ScheduleEveryMonth(ctx context.Context, jobName string, args ...any) (int64, error)

	// ScheduleEveryYear runs a job once a year.
	ScheduleEveryYear(ctx context.Context, jobName string, args ...any) (int64, error)

	// ScheduleInvokeWithTimer looks up a registered job by name and schedules it to run repeatedly
	// based on the given expression, using a lightweight timer.
	ScheduleInvokeWithTimer(ctx context.Context, jobName string, expression string, args ...any) error

	// ScheduleFuncWithTimer runs a custom function on a schedule using timers.
	ScheduleFuncWithTimer(ctx context.Context, expression string, fn func(args ...any) error, args ...any) error

	// ShutDown listens for system interrupt or termination signals (SIGINT, SIGTERM)
	// and performs a graceful shutdown of the JobManagerService by closing
	// the CronJobStore and EnqueuedJobStore resources.
	// It blocks execution until one of the specified signals is received,
	// then releases resources and logs shutdown progress.
	ShutDown()
}

type JobManagerService struct {
	EnqueuedJobStore store.EnqueuedJobStore
	CronJobStore     store.CronJobStore
	MBroker          message_broaker.MessageBroker
	JobHandler       JobHandler
	lockManager      lock.DistributedLockManager
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	writeJobsToQueue bool
	jobQueueName     string
}

func NewJobManager(enqueuedStore store.EnqueuedJobStore, cronStore store.CronJobStore, jobHandler JobHandler, lockManager lock.DistributedLockManager, messageBroker message_broaker.MessageBroker, writeJobsToQueue bool, jobQueueName string) *JobManagerService {
	return &JobManagerService{
		EnqueuedJobStore: enqueuedStore,
		lockManager:      lockManager,
		CronJobStore:     cronStore,
		JobHandler:       jobHandler,
		MBroker:          messageBroker,
		writeJobsToQueue: writeJobsToQueue,
		jobQueueName:     jobQueueName,
	}
}

func (jm *JobManagerService) Enqueue(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
	if !jm.writeJobsToQueue {
		return jm.EnqueuedJobStore.Insert(ctx, jobName, enqueueAt, args...)
	}

	job := models.Job{
		Name:        jobName,
		Args:        args,
		ScheduledAt: enqueueAt,
	}

	payload, err := json2.Marshal(job)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := jm.MBroker.Publish(jm.jobQueueName, payload); err != nil {
		return 0, fmt.Errorf("failed to publish job to broker: %w", err)
	}

	return 0, nil
}

func (jm *JobManagerService) RemoveEnqueue(ctx context.Context, jobID int64) error {
	return jm.EnqueuedJobStore.RemoveByID(ctx, jobID)
}

func (jm *JobManagerService) FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error) {
	return jm.EnqueuedJobStore.FindByID(ctx, jobID)
}

func (jm *JobManagerService) Schedule(ctx context.Context, jobName string, expression string, args ...any) (int64, error) {
	return jm.addOrUpdate(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ScheduleEveryMinute(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "* * * * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ScheduleEveryHour(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 * * * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ScheduleEveryDay(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 * * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ScheduleEveryWeek(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 * * 0"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ScheduleEveryMonth(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 1 * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ScheduleEveryYear(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 1 1 *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

func (jm *JobManagerService) ActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobStore.Activate(ctx, jobID)
}

func (jm *JobManagerService) DeActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobStore.DeActivate(ctx, jobID)
}

func (jm *JobManagerService) ScheduleInvokeWithTimer(ctx context.Context, jobName string, expression string, args ...any) error {
	runJob := func(args ...any) error {
		if !jm.JobHandler.Exists(jobName) {
			return fmt.Errorf("handler for '%s' not found", jobName)
		}
		return jm.JobHandler.Execute(jobName, args...)
	}

	go func() {
		err := jm.runWithTimerInternal(ctx, expression, runJob, args...)
		if err != nil {
			log.Printf("ScheduleInvokeWithTimer: job '%s' stopped: %v", jobName, err)
		}
	}()

	return nil
}

func (jm *JobManagerService) ScheduleFuncWithTimer(ctx context.Context, expression string, fn func(args ...any) error, args ...any) error {
	return jm.runWithTimerInternal(ctx, expression, fn, args...)
}

func (jm *JobManagerService) ShutDown() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Wait for shutdown signal
	<-ctx.Done()

	log.Println("Gofire Shutting down gracefully...")

	// Cancel background job processors
	if jm.cancel != nil {
		jm.cancel()
	}

	// Wait for all background job processors to finish
	jm.wg.Wait()

	if err := jm.CronJobStore.Close(); err != nil {
		log.Println(err.Error())
	}

	if err := jm.EnqueuedJobStore.Close(); err != nil {
		log.Println(err.Error())
	}

	for _, lockID := range constants.Locks {
		jm.lockManager.Release(lockID)
	}

	log.Println("Gofire Shutdown complete.")
}

func (jm *JobManagerService) runWithTimerInternal(ctx context.Context, expression string, fn func(args ...any) error, args ...any) error {
	for {
		next := parser.CalculateNextRun(expression, time.Now())
		duration := time.Until(next)

		if duration <= 0 {
			err := fn(args...)
			if err != nil {
				log.Printf("Job execution error: %v", err)
			}
			continue
		}

		timer := time.NewTimer(duration)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			err := fn(args...)
			if err != nil {
				log.Printf("Job execution error: %v", err)
			}
		}
	}
}

func (jm *JobManagerService) addOrUpdate(ctx context.Context, jobName string, expression string, args ...any) (int64, error) {
	scheduledAt := parser.CalculateNextRun(expression, time.Now())
	return jm.CronJobStore.AddOrUpdate(ctx, jobName, scheduledAt, expression, args...)
}
