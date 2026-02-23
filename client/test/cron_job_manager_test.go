package test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/client/test/mocks"
	"github.com/RezaEskandarii/gofire/types"
	"github.com/RezaEskandarii/gofire/types/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewCronJobManager(t *testing.T) {
	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	cm := client.NewCronJobManager(cronStore, lockMgr, jobHandler, "test-instance")
	assert.NotNil(t, cm)
	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)
}

func TestCronJobManager_Start_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	cronStore.FetchDueCronJobsFunc = func(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error) {
		return &types.PaginationResult[types.CronJob]{
			Items:       []types.CronJob{},
			HasNextPage: false,
		}, nil
	}

	cm := client.NewCronJobManager(cronStore, lockMgr, jobHandler, "test-instance")

	done := make(chan error, 1)
	go func() {
		done <- cm.Start(ctx, 1, 2, 10)
	}()

	// Cancel after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-done
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCronJobManager_Start_ProcessesDueJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	payload, _ := json.Marshal([]any{"arg1"})
	dueJob := types.CronJob{
		ID:         1,
		Name:       "TestCronJob",
		Payload:    payload,
		Expression: "* * * * *",
		NextRunAt:  time.Now().Add(-time.Hour),
	}

	var lockCalled, markSuccessCalled bool
	cronStore.FetchDueCronJobsFunc = func(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error) {
		if page == 1 {
			return &types.PaginationResult[types.CronJob]{
				Items:       []types.CronJob{dueJob},
				HasNextPage: false,
			}, nil
		}
		return &types.PaginationResult[types.CronJob]{Items: []types.CronJob{}, HasNextPage: false}, nil
	}
	cronStore.LockJobFunc = func(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
		lockCalled = true
		return true, nil
	}
	cronStore.MarkSuccessFunc = func(ctx context.Context, jobID int64) error {
		markSuccessCalled = true
		return nil
	}
	cronStore.UpdateJobRunTimesFunc = func(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
		return nil
	}

	_ = jobHandler.Register("TestCronJob", func(args ...any) error {
		return nil
	})

	cm := client.NewCronJobManager(cronStore, lockMgr, jobHandler, "test-instance")

	done := make(chan error, 1)
	go func() {
		done <- cm.Start(ctx, 1, 2, 10)
	}()

	// Allow processing
	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	assert.True(t, lockCalled)
	assert.True(t, markSuccessCalled)
}

func TestCronJobManager_Start_HandlesHandlerNotFound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	payload, _ := json.Marshal([]any{})
	dueJob := types.CronJob{
		ID:         2,
		Name:       "NonExistentHandler",
		Payload:    payload,
		Expression: "* * * * *",
	}

	var markFailureCalled bool
	cronStore.FetchDueCronJobsFunc = func(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error) {
		return &types.PaginationResult[types.CronJob]{
			Items:       []types.CronJob{dueJob},
			HasNextPage: false,
		}, nil
	}
	cronStore.LockJobFunc = func(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
		return true, nil
	}
	cronStore.MarkFailureFunc = func(ctx context.Context, jobID int64, errMsg string) error {
		markFailureCalled = true
		assert.Contains(t, errMsg, "handler not found")
		return nil
	}
	cronStore.UpdateJobRunTimesFunc = func(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
		return nil
	}

	cm := client.NewCronJobManager(cronStore, lockMgr, jobHandler, "test-instance")

	done := make(chan error, 1)
	go func() {
		done <- cm.Start(ctx, 1, 2, 10)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	assert.True(t, markFailureCalled)
}

func TestCronJobManager_Start_HandlesFetchError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	cronStore.FetchDueCronJobsFunc = func(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error) {
		return nil, errors.New("fetch failed")
	}

	cm := client.NewCronJobManager(cronStore, lockMgr, jobHandler, "test-instance")

	done := make(chan error, 1)
	go func() {
		done <- cm.Start(ctx, 1, 2, 10)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	err := <-done
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestCronJobManager_Start_HandlesInvalidPayload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	_ = jobHandler.Register("ValidJob", func(args ...any) error { return nil })

	dueJob := types.CronJob{
		ID:         3,
		Name:       "ValidJob",
		Payload:    json.RawMessage(`invalid json`),
		Expression: "* * * * *",
	}

	var markFailureCalled bool
	cronStore.FetchDueCronJobsFunc = func(ctx context.Context, page int, pageSize int) (*types.PaginationResult[types.CronJob], error) {
		return &types.PaginationResult[types.CronJob]{
			Items:       []types.CronJob{dueJob},
			HasNextPage: false,
		}, nil
	}
	cronStore.LockJobFunc = func(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
		return true, nil
	}
	cronStore.MarkFailureFunc = func(ctx context.Context, jobID int64, errMsg string) error {
		markFailureCalled = true
		assert.Contains(t, errMsg, "invalid payload")
		return nil
	}
	cronStore.UpdateJobRunTimesFunc = func(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
		return nil
	}

	cm := client.NewCronJobManager(cronStore, lockMgr, jobHandler, "test-instance")

	done := make(chan error, 1)
	go func() {
		done <- cm.Start(ctx, 1, 2, 10)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	assert.True(t, markFailureCalled)
}
