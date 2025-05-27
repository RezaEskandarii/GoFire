package gofire

import (
	"context"
	"gofire/internal/models"
	"gofire/internal/parser"
	"gofire/internal/repository"
	"log"
	"time"
)

type JobManager interface {
	Enqueue(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error)
	RemoveEnqueue(ctx context.Context, jobID int64) error
	FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error)
	Schedule(ctx context.Context, jobName string, args []interface{}, expression string) (int64, error)
	ActivateSchedule(ctx context.Context, jobID int64)
	DeActivateSchedule(ctx context.Context, jobID int64)
	ScheduleEveryMinute(ctx context.Context, jobName string, args []interface{}) (int64, error)
	ScheduleEveryHour(ctx context.Context, jobName string, args []interface{}) (int64, error)
	ScheduleEveryDay(ctx context.Context, jobName string, args []interface{}) (int64, error)
	ScheduleEveryWeek(ctx context.Context, jobName string, args []interface{}) (int64, error)
	ScheduleEveryMonth(ctx context.Context, jobName string, args []interface{}) (int64, error)
	ScheduleEveryYear(ctx context.Context, jobName string, args []interface{}) (int64, error)
	FireAndForget(jobName string, args []interface{})
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

func (jm JobManagerService) Enqueue(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error) {
	return jm.EnqueuedJobRepository.Insert(ctx, jobName, scheduledAt, args)
}

func (jm JobManagerService) RemoveEnqueue(ctx context.Context, jobID int64) error {
	return jm.EnqueuedJobRepository.RemoveByID(ctx, jobID)
}

func (jm JobManagerService) FindEnqueue(ctx context.Context, jobID int64) (*models.EnqueuedJob, error) {
	return jm.EnqueuedJobRepository.FindByID(ctx, jobID)
}

func (jm JobManagerService) Schedule(ctx context.Context, jobName string, args []interface{}, expression string) (int64, error) {
	scheduledAt := parser.CalculateNextRun(expression, time.Now())
	return jm.CronJobRepository.AddOrUpdate(ctx, jobName, scheduledAt, args, expression)
}

func (jm JobManagerService) ScheduleEveryMinute(ctx context.Context, jobName string, args []interface{}) (int64, error) {
	return jm.Schedule(ctx, jobName, args, "* * * * *")
}

func (jm JobManagerService) ScheduleEveryHour(ctx context.Context, jobName string, args []interface{}) (int64, error) {
	return jm.Schedule(ctx, jobName, args, "0 * * * *")
}

func (jm JobManagerService) ScheduleEveryDay(ctx context.Context, jobName string, args []interface{}) (int64, error) {
	return jm.Schedule(ctx, jobName, args, "0 0 * * *")
}

func (jm JobManagerService) ScheduleEveryWeek(ctx context.Context, jobName string, args []interface{}) (int64, error) {
	return jm.Schedule(ctx, jobName, args, "0 0 * * 0")
}

func (jm JobManagerService) ScheduleEveryMonth(ctx context.Context, jobName string, args []interface{}) (int64, error) {
	return jm.Schedule(ctx, jobName, args, "0 0 1 * *")
}

func (jm JobManagerService) ScheduleEveryYear(ctx context.Context, jobName string, args []interface{}) (int64, error) {
	return jm.Schedule(ctx, jobName, args, "0 0 1 1 *")
}

func (jm JobManagerService) ActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobRepository.Activate(ctx, jobID)
}

func (jm JobManagerService) DeActivateSchedule(ctx context.Context, jobID int64) {
	jm.CronJobRepository.DeActivate(ctx, jobID)
}

func (jm JobManagerService) FireAndForget(jobName string, args []interface{}) {
	go func() {
		if !jm.JobHandler.Exists(jobName) {
			log.Printf("FireAndForget: handler for '%s' not found", jobName)
			return
		}

		err := jm.JobHandler.Execute(jobName, args)
		if err != nil {
			log.Printf("FireAndForget: job '%s' failed: %v", jobName, err)
		} else {
			log.Printf("FireAndForget: job '%s' executed successfully", jobName)
		}
	}()
}
