package mocks

import (
	"context"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/RezaEskandarii/gofire/types"
	"time"
)

// MockCronJobStore is a mock implementation of store.CronJobStore for testing.
type MockCronJobStore struct {
	AddOrUpdateFunc                 func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error)
	FetchDueCronJobsFunc            func(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error)
	LockJobFunc                     func(ctx context.Context, jobID int64, lockedBy string) (bool, error)
	UnLockJobFunc                   func(ctx context.Context, jobID int64) (bool, error)
	GetAllFunc                      func(ctx context.Context, page int, pageSize int, status state.JobStatus) (*types.PaginationResult[types.CronJob], error)
	CountAllJobsGroupedByStatusFunc func(ctx context.Context) (map[state.JobStatus]int, error)
	UpdateJobRunTimesFunc           func(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error
	MarkSuccessFunc                 func(ctx context.Context, jobID int64) error
	MarkFailureFunc                 func(ctx context.Context, jobID int64, errMsg string) error
	ActivateFunc                    func(ctx context.Context, jobID int64) error
	DeActivateFunc                  func(ctx context.Context, jobID int64) error
	CloseFunc                       func() error
}

func (m *MockCronJobStore) AddOrUpdate(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
	if m.AddOrUpdateFunc != nil {
		return m.AddOrUpdateFunc(ctx, jobName, scheduledAt, expression, args...)
	}
	return 1, nil
}

func (m *MockCronJobStore) FetchDueCronJobs(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error) {
	if m.FetchDueCronJobsFunc != nil {
		return m.FetchDueCronJobsFunc(ctx, page, pageSize)
	}
	return &types.PaginationResult[types.CronJob]{Items: []types.CronJob{}}, nil
}

func (m *MockCronJobStore) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
	if m.LockJobFunc != nil {
		return m.LockJobFunc(ctx, jobID, lockedBy)
	}
	return true, nil
}

func (m *MockCronJobStore) UnLockJob(ctx context.Context, jobID int64) (bool, error) {
	if m.UnLockJobFunc != nil {
		return m.UnLockJobFunc(ctx, jobID)
	}
	return true, nil
}

func (m *MockCronJobStore) GetAll(ctx context.Context, page int, pageSize int, status state.JobStatus) (*types.PaginationResult[types.CronJob], error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx, page, pageSize, status)
	}
	return &types.PaginationResult[types.CronJob]{Items: []types.CronJob{}}, nil
}

func (m *MockCronJobStore) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	if m.CountAllJobsGroupedByStatusFunc != nil {
		return m.CountAllJobsGroupedByStatusFunc(ctx)
	}
	return make(map[state.JobStatus]int), nil
}

func (m *MockCronJobStore) UpdateJobRunTimes(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
	if m.UpdateJobRunTimesFunc != nil {
		return m.UpdateJobRunTimesFunc(ctx, jobID, lastRunAt, nextRunAt)
	}
	return nil
}

func (m *MockCronJobStore) MarkSuccess(ctx context.Context, jobID int64) error {
	if m.MarkSuccessFunc != nil {
		return m.MarkSuccessFunc(ctx, jobID)
	}
	return nil
}

func (m *MockCronJobStore) MarkFailure(ctx context.Context, jobID int64, errMsg string) error {
	if m.MarkFailureFunc != nil {
		return m.MarkFailureFunc(ctx, jobID, errMsg)
	}
	return nil
}

func (m *MockCronJobStore) Activate(ctx context.Context, jobID int64) error {
	if m.ActivateFunc != nil {
		return m.ActivateFunc(ctx, jobID)
	}
	return nil
}

func (m *MockCronJobStore) DeActivate(ctx context.Context, jobID int64) error {
	if m.DeActivateFunc != nil {
		return m.DeActivateFunc(ctx, jobID)
	}
	return nil
}

func (m *MockCronJobStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
