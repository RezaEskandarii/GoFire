package gofire

import (
	"context"
	"fmt"
	"gofire/internal/models"
	"gofire/internal/parser"
	"gofire/internal/repository"
	"log"
	"time"
)

type JobManager interface {
	Enqueue(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error)
	RemoveEnqueue(ctx context.Context, jobID int64) error
	FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error)
	Schedule(ctx context.Context, jobName string, expression string, args ...any) (int64, error)
	ActivateSchedule(ctx context.Context, jobID int64)
	DeActivateSchedule(ctx context.Context, jobID int64)
	ScheduleEveryMinute(ctx context.Context, jobName string, args ...any) (int64, error)
	ScheduleEveryHour(ctx context.Context, jobName string, args ...any) (int64, error)
	ScheduleEveryDay(ctx context.Context, jobName string, args ...any) (int64, error)
	ScheduleEveryWeek(ctx context.Context, jobName string, args ...any) (int64, error)
	ScheduleEveryMonth(ctx context.Context, jobName string, args ...any) (int64, error)
	ScheduleEveryYear(ctx context.Context, jobName string, args ...any) (int64, error)
	ScheduleInvokeWithTimer(ctx context.Context, jobName string, expression string, args ...any) error
	ScheduleFuncWithTimer(ctx context.Context, expression string, fn func(args ...any) error, args ...any) error
}

type JobManagerService struct {
	EnqueuedJobRepository repository.EnqueuedJobRepository
	CronJobRepository     repository.CronJobRepository
	JobHandler            JobHandler
}

func NewJobManager(enqueuedRepo repository.EnqueuedJobRepository, cronRepo repository.CronJobRepository, jobHandler JobHandler) *JobManagerService {
	return &JobManagerService{
		EnqueuedJobRepository: enqueuedRepo,
		CronJobRepository:     cronRepo,
		JobHandler:            jobHandler,
	}
}

func (jm *JobManagerService) Enqueue(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
	return jm.EnqueuedJobRepository.Insert(ctx, jobName, enqueueAt, args...)
}

func (jm *JobManagerService) RemoveEnqueue(ctx context.Context, jobID int64) error {
	return jm.EnqueuedJobRepository.RemoveByID(ctx, jobID)
}

func (jm *JobManagerService) FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error) {
	return jm.EnqueuedJobRepository.FindByID(ctx, jobID)
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
	jm.CronJobRepository.Activate(ctx, jobID)
}

func (jm *JobManagerService) DeActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobRepository.DeActivate(ctx, jobID)
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
	return jm.CronJobRepository.AddOrUpdate(ctx, jobName, scheduledAt, expression, args...)
}
