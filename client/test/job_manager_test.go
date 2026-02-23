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

func newTestJobManager(t *testing.T, enqueueStore *mocks.MockEnqueuedJobStore, cronStore *mocks.MockCronJobStore, lockMgr *mocks.MockDistributedLockManager, broker *mocks.MockMessageBroker, useQueue bool) *client.JobManager {
	jobHandler := config.NewJobHandler()
	return client.NewJobManager(
		enqueueStore,
		cronStore,
		jobHandler,
		lockMgr,
		broker,
		useQueue,
		"test_queue",
	)
}

func TestNewJobManager(t *testing.T) {
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	jobHandler := config.NewJobHandler()

	jm := client.NewJobManager(enqueueStore, cronStore, jobHandler, lockMgr, nil, false, "")
	require.NotNil(t, jm)
	assert.Equal(t, enqueueStore, jm.EnqueuedJobStore)
	assert.Equal(t, cronStore, jm.CronJobStore)
	assert.Equal(t, jobHandler, jm.JobHandler)
}

func TestJobManager_Enqueue_DirectMode(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}

	var insertCalled bool
	var insertJobName string
	var insertArgs []any
	enqueueStore.InsertFunc = func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
		insertCalled = true
		insertJobName = jobName
		insertArgs = args
		return 42, nil
	}

	jm := newTestJobManager(t, enqueueStore, cronStore, lockMgr, nil, false)

	jobID, err := jm.Enqueue(ctx, "TestJob", time.Now().Add(time.Hour), "arg1", "arg2")
	require.NoError(t, err)
	assert.True(t, insertCalled)
	assert.Equal(t, int64(42), jobID)
	assert.Equal(t, "TestJob", insertJobName)
	assert.Equal(t, []any{"arg1", "arg2"}, insertArgs)
}

func TestJobManager_Enqueue_DirectMode_Error(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	enqueueStore.InsertFunc = func(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
		return 0, errors.New("insert failed")
	}

	jm := newTestJobManager(t, enqueueStore, &mocks.MockCronJobStore{}, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.Enqueue(ctx, "TestJob", time.Now(), "arg")
	assert.Error(t, err)
	assert.Equal(t, int64(0), jobID)
	assert.Contains(t, err.Error(), "insert failed")
}

func TestJobManager_Enqueue_QueueMode(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	cronStore := &mocks.MockCronJobStore{}
	lockMgr := &mocks.MockDistributedLockManager{}
	broker := &mocks.MockMessageBroker{}

	var publishedQueue string
	var publishedPayload []byte
	broker.PublishFunc = func(queue string, message []byte) error {
		publishedQueue = queue
		publishedPayload = message
		return nil
	}

	jm := newTestJobManager(t, enqueueStore, cronStore, lockMgr, broker, true)

	jobID, err := jm.Enqueue(ctx, "QueueJob", time.Now().Add(time.Minute), "data")
	require.NoError(t, err)
	assert.Equal(t, int64(0), jobID) // In queue mode, ID is unknown
	assert.Equal(t, "test_queue", publishedQueue)

	var job types.Job
	err = json.Unmarshal(publishedPayload, &job)
	require.NoError(t, err)
	assert.Equal(t, "QueueJob", job.Name)
	assert.Equal(t, []any{"data"}, job.Args)
}

func TestJobManager_Enqueue_QueueMode_PublishError(t *testing.T) {
	ctx := context.Background()
	broker := &mocks.MockMessageBroker{}
	broker.PublishFunc = func(queue string, message []byte) error {
		return errors.New("publish failed")
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, &mocks.MockCronJobStore{}, &mocks.MockDistributedLockManager{}, broker, true)

	jobID, err := jm.Enqueue(ctx, "QueueJob", time.Now(), "data")
	assert.Error(t, err)
	assert.Equal(t, int64(0), jobID)
	assert.Contains(t, err.Error(), "publish failed")
}

func TestJobManager_RemoveEnqueue(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	var removeCalled bool
	var removeID int64
	enqueueStore.RemoveByIDFunc = func(ctx context.Context, jobID int64) error {
		removeCalled = true
		removeID = jobID
		return nil
	}

	jm := newTestJobManager(t, enqueueStore, &mocks.MockCronJobStore{}, &mocks.MockDistributedLockManager{}, nil, false)

	err := jm.RemoveEnqueue(ctx, 99)
	require.NoError(t, err)
	assert.True(t, removeCalled)
	assert.Equal(t, int64(99), removeID)
}

func TestJobManager_RemoveEnqueue_Error(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	enqueueStore.RemoveByIDFunc = func(ctx context.Context, jobID int64) error {
		return errors.New("remove failed")
	}

	jm := newTestJobManager(t, enqueueStore, &mocks.MockCronJobStore{}, &mocks.MockDistributedLockManager{}, nil, false)

	err := jm.RemoveEnqueue(ctx, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remove failed")
}

func TestJobManager_FindEnqueue(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	expectedJob := &types.EnqueuedJob{
		ID:          10,
		Name:        "FindJob",
		Status:      state.StatusQueued,
		ScheduledAt: time.Now(),
	}
	enqueueStore.FindByIDFunc = func(ctx context.Context, id int64) (*types.EnqueuedJob, error) {
		if id == 10 {
			return expectedJob, nil
		}
		return nil, nil
	}

	jm := newTestJobManager(t, enqueueStore, &mocks.MockCronJobStore{}, &mocks.MockDistributedLockManager{}, nil, false)

	job, err := jm.FindEnqueue(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, expectedJob, job)
	assert.Equal(t, int64(10), job.ID)
	assert.Equal(t, "FindJob", job.Name)
}

func TestJobManager_FindEnqueue_Error(t *testing.T) {
	ctx := context.Background()
	enqueueStore := &mocks.MockEnqueuedJobStore{}
	enqueueStore.FindByIDFunc = func(ctx context.Context, id int64) (*types.EnqueuedJob, error) {
		return nil, errors.New("find failed")
	}

	jm := newTestJobManager(t, enqueueStore, &mocks.MockCronJobStore{}, &mocks.MockDistributedLockManager{}, nil, false)

	job, err := jm.FindEnqueue(ctx, 1)
	assert.Error(t, err)
	assert.Nil(t, job)
}

func TestJobManager_Schedule(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addOrUpdateCalled bool
	var addJobName, addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addOrUpdateCalled = true
		addJobName = jobName
		addExpr = expression
		return 7, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.Schedule(ctx, "CronJob", "* * * * *", "arg1")
	require.NoError(t, err)
	assert.True(t, addOrUpdateCalled)
	assert.Equal(t, int64(7), jobID)
	assert.Equal(t, "CronJob", addJobName)
	assert.Equal(t, "* * * * *", addExpr)
}

func TestJobManager_ScheduleEveryMinute(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addExpr = expression
		return 1, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.ScheduleEveryMinute(ctx, "MinuteJob", "arg")
	require.NoError(t, err)
	assert.Equal(t, int64(1), jobID)
	assert.Equal(t, "* * * * *", addExpr)
}

func TestJobManager_ScheduleEveryHour(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addExpr = expression
		return 1, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.ScheduleEveryHour(ctx, "HourJob")
	require.NoError(t, err)
	assert.Equal(t, int64(1), jobID)
	assert.Equal(t, "0 * * * *", addExpr)
}

func TestJobManager_ScheduleEveryDay(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addExpr = expression
		return 1, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.ScheduleEveryDay(ctx, "DayJob")
	require.NoError(t, err)
	assert.Equal(t, int64(1), jobID)
	assert.Equal(t, "0 0 * * *", addExpr)
}

func TestJobManager_ScheduleEveryWeek(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addExpr = expression
		return 1, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.ScheduleEveryWeek(ctx, "WeekJob")
	require.NoError(t, err)
	assert.Equal(t, int64(1), jobID)
	assert.Equal(t, "0 0 * * 0", addExpr)
}

func TestJobManager_ScheduleEveryMonth(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addExpr = expression
		return 1, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.ScheduleEveryMonth(ctx, "MonthJob")
	require.NoError(t, err)
	assert.Equal(t, int64(1), jobID)
	assert.Equal(t, "0 0 1 * *", addExpr)
}

func TestJobManager_ScheduleEveryYear(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var addExpr string
	cronStore.AddOrUpdateFunc = func(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {
		addExpr = expression
		return 1, nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jobID, err := jm.ScheduleEveryYear(ctx, "YearJob")
	require.NoError(t, err)
	assert.Equal(t, int64(1), jobID)
	assert.Equal(t, "0 0 1 1 *", addExpr)
}

func TestJobManager_ActivateSchedule(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var activateCalled bool
	var activateID int64
	cronStore.ActivateFunc = func(ctx context.Context, jobID int64) error {
		activateCalled = true
		activateID = jobID
		return nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jm.ActivateSchedule(ctx, 5)
	assert.True(t, activateCalled)
	assert.Equal(t, int64(5), activateID)
}

func TestJobManager_DeActivateSchedule(t *testing.T) {
	ctx := context.Background()
	cronStore := &mocks.MockCronJobStore{}
	var deactivateCalled bool
	var deactivateID int64
	cronStore.DeActivateFunc = func(ctx context.Context, jobID int64) error {
		deactivateCalled = true
		deactivateID = jobID
		return nil
	}

	jm := newTestJobManager(t, &mocks.MockEnqueuedJobStore{}, cronStore, &mocks.MockDistributedLockManager{}, nil, false)

	jm.DeActivateSchedule(ctx, 3)
	assert.True(t, deactivateCalled)
	assert.Equal(t, int64(3), deactivateID)
}

func TestJobManager_ScheduleInvokeWithTimer(t *testing.T) {
	ctx := context.Background()
	jobHandler := config.NewJobHandler()
	_ = jobHandler.Register("TimerJob", func(args ...any) error {
		return nil
	})

	jm := client.NewJobManager(
		&mocks.MockEnqueuedJobStore{},
		&mocks.MockCronJobStore{},
		jobHandler,
		&mocks.MockDistributedLockManager{},
		nil,
		false,
		"",
	)

	err := jm.ScheduleInvokeWithTimer(ctx, "TimerJob", "* * * * *", "arg")
	require.NoError(t, err)

	// ScheduleInvokeWithTimer starts a goroutine; give it a moment
	time.Sleep(50 * time.Millisecond)
}

func TestJobManager_ScheduleInvokeWithTimer_HandlerNotFound(t *testing.T) {
	ctx := context.Background()
	jobHandler := config.NewJobHandler()

	jm := client.NewJobManager(
		&mocks.MockEnqueuedJobStore{},
		&mocks.MockCronJobStore{},
		jobHandler,
		&mocks.MockDistributedLockManager{},
		nil,
		false,
		"",
	)

	err := jm.ScheduleInvokeWithTimer(ctx, "NonExistentJob", "* * * * *")
	require.NoError(t, err) // Does not return error; runs in goroutine
	time.Sleep(100 * time.Millisecond)
}

func TestJobManager_ScheduleFuncWithTimer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobHandler := config.NewJobHandler()
	jm := client.NewJobManager(
		&mocks.MockEnqueuedJobStore{},
		&mocks.MockCronJobStore{},
		jobHandler,
		&mocks.MockDistributedLockManager{},
		nil,
		false,
		"",
	)

	// ScheduleFuncWithTimer blocks and runs the cron loop; verify it accepts the expression
	// Use a far-future cron to avoid long wait: Dec 31 23:59
	executed := make(chan bool, 1)
	go func() {
		_ = jm.ScheduleFuncWithTimer(ctx, "59 23 31 12 *", func(args ...any) error {
			executed <- true
			return nil
		})
	}()

	// Cancel quickly so the loop exits
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
