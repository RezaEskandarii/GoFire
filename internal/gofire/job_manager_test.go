package gofire

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gofire/internal/models"
	"gofire/internal/state"
	"testing"
	"time"
)

// ===================== EnqueuedJobRepository Mock =========================
type MockEnqueuedJobRepository struct {
	MockFindByID                    func(ctx context.Context, id int64) (*models.EnqueuedJob, error)
	MockRemoveByID                  func(ctx context.Context, jobID int64) error
	MockInsert                      func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error)
	MockFetchDueJobs                func(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error)
	MockLockJob                     func(ctx context.Context, jobID int64, lockedBy string) (bool, error)
	MockMarkSuccess                 func(ctx context.Context, jobID int64) error
	MockMarkFailure                 func(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error
	MockUnlockStaleJobs             func(ctx context.Context, timeout time.Duration) error
	MockCountJobsByStatus           func(ctx context.Context, status state.JobStatus) (int, error)
	MockCountAllJobsGroupedByStatus func(ctx context.Context) (map[state.JobStatus]int, error)
	MockMarkRetryFailedJobs         func(ctx context.Context) error
	MockClose                       func()
}

func (m *MockEnqueuedJobRepository) FindByID(ctx context.Context, id int64) (*models.EnqueuedJob, error) {
	return m.MockFindByID(ctx, id)
}
func (m *MockEnqueuedJobRepository) RemoveByID(ctx context.Context, jobID int64) error {
	return m.MockRemoveByID(ctx, jobID)
}
func (m *MockEnqueuedJobRepository) Insert(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
	return m.MockInsert(ctx, jobName, enqueueAt, args...)
}
func (m *MockEnqueuedJobRepository) FetchDueJobs(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error) {
	return m.MockFetchDueJobs(ctx, page, pageSize, statuses, scheduledBefore)
}
func (m *MockEnqueuedJobRepository) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
	return m.MockLockJob(ctx, jobID, lockedBy)
}
func (m *MockEnqueuedJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	return m.MockMarkSuccess(ctx, jobID)
}
func (m *MockEnqueuedJobRepository) MarkFailure(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error {
	return m.MockMarkFailure(ctx, jobID, errMsg, attempts, maxAttempts)
}
func (m *MockEnqueuedJobRepository) UnlockStaleJobs(ctx context.Context, timeout time.Duration) error {
	return m.MockUnlockStaleJobs(ctx, timeout)
}
func (m *MockEnqueuedJobRepository) CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error) {
	return m.MockCountJobsByStatus(ctx, status)
}
func (m *MockEnqueuedJobRepository) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	return m.MockCountAllJobsGroupedByStatus(ctx)
}
func (m *MockEnqueuedJobRepository) MarkRetryFailedJobs(ctx context.Context) error {
	return m.MockMarkRetryFailedJobs(ctx)
}
func (m *MockEnqueuedJobRepository) Close() {
	m.MockClose()
}

// ===================== CronJobRepository Mock =========================
type MockCronJobRepository struct {
	MockAddOrUpdate                 func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error)
	MockFetchDueCronJobs            func(ctx context.Context, page int, pageSize int) (*models.PaginationResult[models.CronJob], error)
	MockGetAll                      func(ctx context.Context, page int, pageSize int, status state.JobStatus) (*models.PaginationResult[models.CronJob], error)
	MockCountAllJobsGroupedByStatus func(ctx context.Context) (map[state.JobStatus]int, error)
	MockUpdateJobRunTimes           func(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error
	MockMarkSuccess                 func(ctx context.Context, jobID int64) error
	MockMarkFailure                 func(ctx context.Context, jobID int64, errMsg string) error
	MockActivate                    func(ctx context.Context, jobID int64) error
	MockDeActivate                  func(ctx context.Context, jobID int64) error
	MockClose                       func()
}

func (m *MockCronJobRepository) AddOrUpdate(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
	return m.MockAddOrUpdate(ctx, jobName, scheduledAt, expression, args...)
}
func (m *MockCronJobRepository) FetchDueCronJobs(ctx context.Context, page int, pageSize int) (*models.PaginationResult[models.CronJob], error) {
	return m.MockFetchDueCronJobs(ctx, page, pageSize)
}
func (m *MockCronJobRepository) GetAll(ctx context.Context, page int, pageSize int, status state.JobStatus) (*models.PaginationResult[models.CronJob], error) {
	return m.MockGetAll(ctx, page, pageSize, status)
}
func (m *MockCronJobRepository) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	return m.MockCountAllJobsGroupedByStatus(ctx)
}
func (m *MockCronJobRepository) UpdateJobRunTimes(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
	return m.MockUpdateJobRunTimes(ctx, jobID, lastRunAt, nextRunAt)
}
func (m *MockCronJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	return m.MockMarkSuccess(ctx, jobID)
}
func (m *MockCronJobRepository) MarkFailure(ctx context.Context, jobID int64, errMsg string) error {
	return m.MockMarkFailure(ctx, jobID, errMsg)
}
func (m *MockCronJobRepository) Activate(ctx context.Context, jobID int64) error {
	return m.MockActivate(ctx, jobID)
}
func (m *MockCronJobRepository) DeActivate(ctx context.Context, jobID int64) error {
	return m.MockDeActivate(ctx, jobID)
}
func (m *MockCronJobRepository) Close() {
	m.MockClose()
}

// ===================== JobManagerService Unit Tests =========================

func TestJobManagerService_Enqueue(t *testing.T) {
	mockEnqueuedRepo := &MockEnqueuedJobRepository{
		MockInsert: func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
			return 42, nil
		},
	}
	mockCronRepo := &MockCronJobRepository{}
	jm := NewJobManager(mockEnqueuedRepo, mockCronRepo, createTestJobHandler())

	id, err := jm.Enqueue(context.Background(), "email", time.Now())
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestJobManagerService_RemoveEnqueue(t *testing.T) {
	mockEnqueuedRepo := &MockEnqueuedJobRepository{
		MockRemoveByID: func(ctx context.Context, jobID int64) error {
			return nil
		},
	}
	jm := NewJobManager(mockEnqueuedRepo, nil, createTestJobHandler())

	err := jm.RemoveEnqueue(context.Background(), 1)
	assert.NoError(t, err)
}

func TestJobManagerService_FindEnqueue(t *testing.T) {
	exampleJob := &models.EnqueuedJob{ID: 1, Name: "test"}
	mockEnqueuedRepo := &MockEnqueuedJobRepository{
		MockFindByID: func(ctx context.Context, id int64) (*models.EnqueuedJob, error) {
			return exampleJob, nil
		},
	}
	jm := NewJobManager(mockEnqueuedRepo, nil, createTestJobHandler())

	job, err := jm.FindEnqueue(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, exampleJob, job)
}

func TestJobManagerService_Schedule(t *testing.T) {
	mockCronRepo := &MockCronJobRepository{
		MockAddOrUpdate: func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
			return 99, nil
		},
	}
	jm := NewJobManager(nil, mockCronRepo, createTestJobHandler())

	id, err := jm.Schedule(context.Background(), "daily", "0 0 * * *")
	assert.NoError(t, err)
	assert.Equal(t, int64(99), id)
}

func TestJobManagerService_ActivateSchedule(t *testing.T) {
	var activatedID int64
	mockCronRepo := &MockCronJobRepository{
		MockActivate: func(ctx context.Context, jobID int64) error {
			activatedID = jobID
			return nil
		},
	}
	jm := NewJobManager(nil, mockCronRepo, createTestJobHandler())

	jm.ActivateSchedule(context.Background(), 123)
	assert.Equal(t, int64(123), activatedID)
}

func TestJobManagerService_DeActivateSchedule(t *testing.T) {
	var deactivatedID int64
	mockCronRepo := &MockCronJobRepository{
		MockDeActivate: func(ctx context.Context, jobID int64) error {
			deactivatedID = jobID
			return nil
		},
	}
	jm := NewJobManager(nil, mockCronRepo, createTestJobHandler())

	jm.DeActivateSchedule(context.Background(), 456)
	assert.Equal(t, int64(456), deactivatedID)
}

// Similar pattern can be used to test ScheduleEveryMinute, Hour, Day, etc.
func TestJobManagerService_ScheduleEveryMinute(t *testing.T) {
	mockCronRepo := &MockCronJobRepository{
		MockAddOrUpdate: func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
			assert.Equal(t, "* * * * *", expression)
			return 7, nil
		},
	}
	jm := NewJobManager(nil, mockCronRepo, createTestJobHandler())

	id, err := jm.ScheduleEveryMinute(context.Background(), "heartbeat")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), id)
}

func createTestJobHandler() JobHandler {
	jh := NewJobHandler()
	_ = jh.Register("email", func(args ...any) error {
		return nil
	})
	_ = jh.Register("daily", func(args ...any) error {
		return nil
	})
	_ = jh.Register("heartbeat", func(args ...any) error {
		return nil
	})
	_ = jh.Register("test", func(args ...any) error {
		return nil
	})
	return jh
}
