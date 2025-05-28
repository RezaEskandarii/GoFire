package repository

import (
	"context"
	"gofire/internal/models"
	"time"
)

// CronJobRepository defines the interface for managing Cron Jobs in DB.
type CronJobRepository interface {
	// AddOrUpdate inserts a new cron job or updates its scheduled time and arguments if it already exists.
	// Returns the job's ID.
	AddOrUpdate(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error)

	// FetchDueCronJobs fetches active cron jobs whose NextRunAt <= now, limited by 'limit'.
	FetchDueCronJobs(ctx context.Context, page int, pageSize int) (*models.PaginationResult[models.CronJob], error)

	// UpdateJobRunTimes updates the LastRunAt and NextRunAt timestamps after execution.
	UpdateJobRunTimes(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error

	// MarkSuccess marks the job as successfully executed.
	MarkSuccess(ctx context.Context, jobID int64) error

	// MarkFailure marks the job execution as failed with the given error message.
	MarkFailure(ctx context.Context, jobID int64, errMsg string) error

	// Activate Job
	Activate(ctx context.Context, jobID int64) error

	// Deactivate Job
	DeActivate(ctx context.Context, jobID int64) error
}
