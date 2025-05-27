package repository

import (
	"context"
	"gofire/internal/models"
	"gofire/internal/state"
	"time"
)

type EnqueuedJobRepository interface {
	FindByID(ctx context.Context, id int64) (*models.EnqueuedJob, error)
	RemoveByID(ctx context.Context, jobID int64) error
	Insert(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error)
	FetchDueJobs(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error)
	LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error)
	MarkSuccess(ctx context.Context, jobID int64) error
	MarkFailure(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error
	UnlockStaleJobs(ctx context.Context, timeout time.Duration) error
	CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error)
	CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error)
	MarkRetryFailedJobs(ctx context.Context) error
}
