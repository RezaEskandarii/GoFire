package gofire

import (
	"context"
	json2 "encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/lock"
	"gofire/internal/message_broaker"
	"gofire/internal/models"
	"gofire/internal/store"
	"gofire/pgk/parser"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type JobManager struct {
	EnqueuedJobStore store.EnqueuedJobStore
	CronJobStore     store.CronJobStore
	MBroker          message_broaker.MessageBroker
	JobHandler       *JobHandler
	lockManager      lock.DistributedLockManager
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	writeJobsToQueue bool
	jobQueueName     string
}

func NewJobManager(enqueuedStore store.EnqueuedJobStore, cronStore store.CronJobStore, jobHandler *JobHandler, lockManager lock.DistributedLockManager, messageBroker message_broaker.MessageBroker, writeJobsToQueue bool, jobQueueName string) *JobManager {
	return &JobManager{
		EnqueuedJobStore: enqueuedStore,
		lockManager:      lockManager,
		CronJobStore:     cronStore,
		JobHandler:       jobHandler,
		MBroker:          messageBroker,
		writeJobsToQueue: writeJobsToQueue,
		jobQueueName:     jobQueueName,
	}
}

// Enqueue either directly stores the job in the database (if disabled queue mode),
// or publishes it to a message broker like RabbitMQ if enabled.
// In queue mode, it returns 0 as job ID is unknown at this stage.
func (jm *JobManager) Enqueue(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
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

// RemoveEnqueue deletes a queued job using its ID.
func (jm *JobManager) RemoveEnqueue(ctx context.Context, jobID int64) error {
	return jm.EnqueuedJobStore.RemoveByID(ctx, jobID)
}

// FindEnqueue returns the details of a queued job by its ID.
func (jm *JobManager) FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error) {
	return jm.EnqueuedJobStore.FindByID(ctx, jobID)
}

// Schedule sets up a recurring job based on a cron expression.
func (jm *JobManager) Schedule(ctx context.Context, jobName string, expression string, args ...any) (int64, error) {
	return jm.addOrUpdate(ctx, jobName, expression, args...)
}

// ScheduleEveryMinute runs a job once every minute.
func (jm *JobManager) ScheduleEveryMinute(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "* * * * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

// ScheduleEveryHour runs a job once every hour.
func (jm *JobManager) ScheduleEveryHour(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 * * * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

// ScheduleEveryDay runs a job once every day.
func (jm *JobManager) ScheduleEveryDay(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 * * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

// ScheduleEveryWeek runs a job once a week.
func (jm *JobManager) ScheduleEveryWeek(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 * * 0"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

// ScheduleEveryMonth runs a job once a month.
func (jm *JobManager) ScheduleEveryMonth(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 1 * *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

// ScheduleEveryYear runs a job once a year.
func (jm *JobManager) ScheduleEveryYear(ctx context.Context, jobName string, args ...any) (int64, error) {
	expression := "0 0 1 1 *"
	if _, err := jm.addOrUpdate(ctx, jobName, expression, args...); err != nil {
		return 0, err
	}
	return jm.Schedule(ctx, jobName, expression, args...)
}

// ActivateSchedule enables a scheduled job if it was previously disabled.
func (jm *JobManager) ActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobStore.Activate(ctx, jobID)
}

// DeActivateSchedule temporarily disables a scheduled job.
func (jm *JobManager) DeActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobStore.DeActivate(ctx, jobID)
}

// ScheduleInvokeWithTimer looks up a registered job by name and schedules it to run repeatedly
// based on the given expression, using a lightweight timer.
func (jm *JobManager) ScheduleInvokeWithTimer(ctx context.Context, jobName string, expression string, args ...any) error {
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

// ScheduleFuncWithTimer runs a custom function on a schedule using timers.
func (jm *JobManager) ScheduleFuncWithTimer(ctx context.Context, expression string, fn func(args ...any) error, args ...any) error {
	return jm.runWithTimerInternal(ctx, expression, fn, args...)
}

// GracefulExit listens for system interrupt or termination signals (SIGINT, SIGTERM)
// and performs a graceful shutdown of the JobManager by closing
// the CronJobStore and EnqueuedJobStore resources.
// It blocks execution until one of the specified signals is received,
// then releases resources and logs shutdown progress.
func (jm *JobManager) GracefulExit() {
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

func (jm *JobManager) runWithTimerInternal(ctx context.Context, expression string, fn func(args ...any) error, args ...any) error {
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

func (jm *JobManager) addOrUpdate(ctx context.Context, jobName string, expression string, args ...any) (int64, error) {
	scheduledAt := parser.CalculateNextRun(expression, time.Now())
	return jm.CronJobStore.AddOrUpdate(ctx, jobName, scheduledAt, expression, args...)
}
