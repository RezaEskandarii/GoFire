package repository

import (
	"context"
	"gofire/internal/models"
	"gofire/internal/state"
	"time"
)

type EnqueuedJobRepository interface {
	// FindByID retrieves an enqueued job by its unique ID.
	FindByID(ctx context.Context, id int64) (*models.EnqueuedJob, error)

	// RemoveByID deletes the job identified by jobID from the queue.
	RemoveByID(ctx context.Context, jobID int64) error

	// Insert adds a new job to the queue with the specified name, scheduled time, and optional arguments.
	// Returns the ID of the newly inserted job.
	Insert(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error)

	// FetchDueJobs retrieves a paginated list of jobs that are due to run, filtered by their statuses and scheduled time.
	FetchDueJobs(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error)

	// LockJob attempts to lock the job for processing by the given worker identified by lockedBy.
	// Returns true if the lock was acquired successfully.
	LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error)

	// MarkSuccess marks the job as successfully completed.
	MarkSuccess(ctx context.Context, jobID int64) error

	// MarkFailure records a failed attempt for the job, including an error message and tracking attempt counts.
	MarkFailure(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error

	// UnlockStaleJobs releases locks on jobs that have been locked longer than the specified timeout duration.
	UnlockStaleJobs(ctx context.Context, timeout time.Duration) error

	// CountJobsByStatus returns the number of jobs currently in a given status.
	CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error)

	// CountAllJobsGroupedByStatus returns counts of all jobs grouped by their status.
	CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error)

	// MarkRetryFailedJobs flags jobs that have failed but are eligible to be retried.
	MarkRetryFailedJobs(ctx context.Context) error

	Close()
}
