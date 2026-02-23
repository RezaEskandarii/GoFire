package test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/client/test/mocks"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/RezaEskandarii/gofire/types"
	"github.com/RezaEskandarii/gofire/types/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func newTestEnqueueManager(t *testing.T, store *mocks.MockEnqueuedJobStore, lockMgr *mocks.MockDistributedLockManager, broker *mocks.MockMessageBroker) client.EnqueueJobsManager {
	jobHandler := config.NewJobHandler()
	em := client.NewEnqueueScheduler(store, lockMgr, jobHandler, broker, "test-instance")
	// UnlockStaleJobs is called in Start - must succeed
	if store.UnlockStaleJobsFunc == nil {
		store.UnlockStaleJobsFunc = func(ctx context.Context, timeout time.Duration) error {
			return nil
		}
	}
	return em
}

func TestNewEnqueueScheduler(t *testing.T) {
	store := &mocks.MockEnqueuedJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	em := client.NewEnqueueScheduler(store, lockMgr, jobHandler, nil, "instance-1")
	assert.NotNil(t, em)
	time.Sleep(50 * time.Millisecond)
}

func TestEnqueueJobsManager_Enqueue(t *testing.T) {
	ctx := context.Background()
	store := &mocks.MockEnqueuedJobStore{}
	var insertCalled bool
	var insertName string
	store.InsertFunc = func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
		insertCalled = true
		insertName = jobName
		return 100, nil
	}

	em := newTestEnqueueManager(t, store, &mocks.MockDistributedLockManager{}, nil)

	jobID, err := em.Enqueue(ctx, "EnqueueJob", time.Now().Add(time.Hour), "a", "b")
	require.NoError(t, err)
	assert.True(t, insertCalled)
	assert.Equal(t, int64(100), jobID)
	assert.Equal(t, "EnqueueJob", insertName)
}

func TestEnqueueJobsManager_Enqueue_Error(t *testing.T) {
	ctx := context.Background()
	store := &mocks.MockEnqueuedJobStore{}
	store.InsertFunc = func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
		return 0, errors.New("insert error")
	}

	em := newTestEnqueueManager(t, store, &mocks.MockDistributedLockManager{}, nil)

	jobID, err := em.Enqueue(ctx, "Job", time.Now(), "arg")
	assert.Error(t, err)
	assert.Equal(t, int64(0), jobID)
	assert.Contains(t, err.Error(), "insert error")
}

func TestEnqueueJobsManager_Start_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &mocks.MockEnqueuedJobStore{}
	store.UnlockStaleJobsFunc = func(ctx context.Context, timeout time.Duration) error {
		return nil
	}
	store.FetchDueJobsFunc = func(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*types.PaginationResult[types.EnqueuedJob], error) {
		return &types.PaginationResult[types.EnqueuedJob]{
			Items:       []types.EnqueuedJob{},
			HasNextPage: false,
		}, nil
	}

	em := newTestEnqueueManager(t, store, &mocks.MockDistributedLockManager{}, nil)

	done := make(chan error, 1)
	go func() {
		done <- em.Start(ctx, 1, 2, 10)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	err := <-done
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestEnqueueJobsManager_Start_ProcessesDueJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &mocks.MockEnqueuedJobStore{}
	store.UnlockStaleJobsFunc = func(ctx context.Context, timeout time.Duration) error {
		return nil
	}

	payload, _ := json.Marshal([]any{"arg1"})
	dueJob := types.EnqueuedJob{
		ID:          1,
		Name:        "ProcessJob",
		Payload:     payload,
		Status:      state.StatusQueued,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Hour),
		CreatedAt:   time.Now(),
	}

	jobHandler := config.NewJobHandler()
	_ = jobHandler.Register("ProcessJob", func(args ...any) error {
		return nil
	})

	var lockCalled, markSuccessCalled bool
	store.FetchDueJobsFunc = func(ctx context.Context, page int, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*types.PaginationResult[types.EnqueuedJob], error) {
		return &types.PaginationResult[types.EnqueuedJob]{
			Items:       []types.EnqueuedJob{dueJob},
			HasNextPage: false,
		}, nil
	}
	store.LockJobFunc = func(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
		lockCalled = true
		return true, nil
	}
	store.MarkSuccessFunc = func(ctx context.Context, jobID int64) error {
		markSuccessCalled = true
		return nil
	}

	em := client.NewEnqueueScheduler(store, &mocks.MockDistributedLockManager{}, jobHandler, nil, "test-instance")

	done := make(chan error, 1)
	go func() {
		done <- em.Start(ctx, 1, 2, 10)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	assert.True(t, lockCalled)
	assert.True(t, markSuccessCalled)
}

func TestEnqueueJobsManager_ExecuteJobManually(t *testing.T) {
	ctx := context.Background()
	store := &mocks.MockEnqueuedJobStore{}
	jobHandler := config.NewJobHandler()
	_ = jobHandler.Register("ManualJob", func(args ...any) error {
		assert.Equal(t, []any{"arg1"}, args)
		return nil
	})

	payload, _ := json.Marshal([]any{"arg1"})
	job := &types.EnqueuedJob{
		ID:          5,
		Name:        "ManualJob",
		Payload:     payload,
		Status:      state.StatusQueued,
		Attempts:    0,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	store.FindByIDFunc = func(ctx context.Context, id int64) (*types.EnqueuedJob, error) {
		if id == 5 {
			return job, nil
		}
		return nil, nil
	}

	em := client.NewEnqueueScheduler(store, &mocks.MockDistributedLockManager{}, jobHandler, nil, "test")
	store.UnlockStaleJobsFunc = func(ctx context.Context, timeout time.Duration) error { return nil }

	err := em.ExecuteJobManually(ctx, 5)
	require.NoError(t, err)
}

func TestEnqueueJobsManager_ExecuteJobManually_JobNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mocks.MockEnqueuedJobStore{}
	store.FindByIDFunc = func(ctx context.Context, id int64) (*types.EnqueuedJob, error) {
		return nil, errors.New("not found")
	}

	em := newTestEnqueueManager(t, store, &mocks.MockDistributedLockManager{}, nil)

	err := em.ExecuteJobManually(ctx, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEnqueueJobsManager_ExecuteJobManually_HandlerNotFound(t *testing.T) {
	ctx := context.Background()
	store := &mocks.MockEnqueuedJobStore{}
	jobHandler := config.NewJobHandler()

	payload, _ := json.Marshal([]any{})
	job := &types.EnqueuedJob{
		ID:      6,
		Name:    "NoHandlerJob",
		Payload: payload,
	}

	store.FindByIDFunc = func(ctx context.Context, id int64) (*types.EnqueuedJob, error) {
		return job, nil
	}

	em := client.NewEnqueueScheduler(store, &mocks.MockDistributedLockManager{}, jobHandler, nil, "test")
	store.UnlockStaleJobsFunc = func(ctx context.Context, timeout time.Duration) error { return nil }

	err := em.ExecuteJobManually(ctx, 6)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler not found")
}

func TestEnqueueJobsManager_StartQueueAndStorageSyncWorker_NoOpWhenDisabled(t *testing.T) {
	ctx := context.Background()
	store := &mocks.MockEnqueuedJobStore{}
	em := newTestEnqueueManager(t, store, &mocks.MockDistributedLockManager{}, nil)

	err := em.StartQueueAndStorageSyncWorker(ctx, "queue", false)
	require.NoError(t, err)
}

func TestEnqueueJobsManager_StartQueueAndStorageSyncWorker_StartsWhenEnabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &mocks.MockEnqueuedJobStore{}
	broker := &mocks.MockMessageBroker{}

	var consumeCalled bool
	msgCh := make(chan []byte, 1)
	close(msgCh)
	broker.ConsumeFunc = func(ctx context.Context, queue string) (<-chan []byte, error) {
		consumeCalled = true
		return msgCh, nil
	}

	em := newTestEnqueueManager(t, store, &mocks.MockDistributedLockManager{}, broker)

	err := em.StartQueueAndStorageSyncWorker(ctx, "test_queue", true)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.True(t, consumeCalled)
}
