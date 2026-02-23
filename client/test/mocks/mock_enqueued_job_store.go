package mocks

import (
	"context"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/RezaEskandarii/gofire/types"
	"time"
)

// MockEnqueuedJobStore is a mock implementation of store.EnqueuedJobStore for testing.
type MockEnqueuedJobStore struct {
	InsertFunc                      func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error)
	FindByIDFunc                    func(ctx context.Context, id int64) (*types.EnqueuedJob, error)
	RemoveByIDFunc                  func(ctx context.Context, jobID int64) error
	FetchDueJobsFunc                func(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*types.PaginationResult[types.EnqueuedJob], error)
	LockJobFunc                     func(ctx context.Context, jobID int64, lockedBy string) (bool, error)
	MarkSuccessFunc                 func(ctx context.Context, jobID int64) error
	MarkFailureFunc                 func(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error
	UnlockStaleJobsFunc             func(ctx context.Context, timeout time.Duration) error
	CountJobsByStatusFunc           func(ctx context.Context, status state.JobStatus) (int, error)
	CountAllJobsGroupedByStatusFunc func(ctx context.Context) (map[state.JobStatus]int, error)
	MarkRetryFailedJobsFunc         func(ctx context.Context) error
	BulkInsertFunc                  func(ctx context.Context, batch []types.Job) error
	CloseFunc                       func() error
}

func (m *MockEnqueuedJobStore) Insert(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
	if m.InsertFunc != nil {
		return m.InsertFunc(ctx, jobName, enqueueAt, args...)
	}
	return 0, nil
}

func (m *MockEnqueuedJobStore) FindByID(ctx context.Context, id int64) (*types.EnqueuedJob, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEnqueuedJobStore) RemoveByID(ctx context.Context, jobID int64) error {
	if m.RemoveByIDFunc != nil {
		return m.RemoveByIDFunc(ctx, jobID)
	}
	return nil
}

func (m *MockEnqueuedJobStore) FetchDueJobs(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*types.PaginationResult[types.EnqueuedJob], error) {
	if m.FetchDueJobsFunc != nil {
		return m.FetchDueJobsFunc(ctx, page, pageSize, statuses, scheduledBefore)
	}
	return &types.PaginationResult[types.EnqueuedJob]{Items: []types.EnqueuedJob{}}, nil
}

func (m *MockEnqueuedJobStore) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
	if m.LockJobFunc != nil {
		return m.LockJobFunc(ctx, jobID, lockedBy)
	}
	return true, nil
}

func (m *MockEnqueuedJobStore) MarkSuccess(ctx context.Context, jobID int64) error {
	if m.MarkSuccessFunc != nil {
		return m.MarkSuccessFunc(ctx, jobID)
	}
	return nil
}

func (m *MockEnqueuedJobStore) MarkFailure(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error {
	if m.MarkFailureFunc != nil {
		return m.MarkFailureFunc(ctx, jobID, errMsg, attempts, maxAttempts)
	}
	return nil
}

func (m *MockEnqueuedJobStore) UnlockStaleJobs(ctx context.Context, timeout time.Duration) error {
	if m.UnlockStaleJobsFunc != nil {
		return m.UnlockStaleJobsFunc(ctx, timeout)
	}
	return nil
}

func (m *MockEnqueuedJobStore) CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error) {
	if m.CountJobsByStatusFunc != nil {
		return m.CountJobsByStatusFunc(ctx, status)
	}
	return 0, nil
}

func (m *MockEnqueuedJobStore) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	if m.CountAllJobsGroupedByStatusFunc != nil {
		return m.CountAllJobsGroupedByStatusFunc(ctx)
	}
	return make(map[state.JobStatus]int), nil
}

func (m *MockEnqueuedJobStore) MarkRetryFailedJobs(ctx context.Context) error {
	if m.MarkRetryFailedJobsFunc != nil {
		return m.MarkRetryFailedJobsFunc(ctx)
	}
	return nil
}

func (m *MockEnqueuedJobStore) BulkInsert(ctx context.Context, batch []types.Job) error {
	if m.BulkInsertFunc != nil {
		return m.BulkInsertFunc(ctx, batch)
	}
	return nil
}

func (m *MockEnqueuedJobStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
