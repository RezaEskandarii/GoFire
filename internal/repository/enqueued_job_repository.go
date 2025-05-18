package repository

import (
	"context"
	"gofire/internal/models"
	"gofire/internal/state"
	"time"
)

type EnqueuedJobRepository interface {
	Insert(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error)
	FetchDueJobs(ctx context.Context, page int, pageSize int, status *state.JobStatus, scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error)
	LockJob(ctx context.Context, job *models.EnqueuedJob, lockedBy string) (bool, error)
	MarkSuccess(ctx context.Context, jobID int) error
	MarkFailure(ctx context.Context, jobID int, errMsg string, attempts int, maxAttempts int) error
	UnlockStaleJobs(ctx context.Context, timeout time.Duration) error
	CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error)
	CountAllJobsByStatus(ctx context.Context) (map[state.JobStatus]int, error)
}
