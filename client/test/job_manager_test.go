package test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestJobManager_Enqueue(t *testing.T) {
	id, err := testJobManager.Enqueue(context.Background(), "email", time.Now(), "arg1", "arg2")
	assert.NoError(t, err)
	assert.True(t, id >= 0)
}

func TestJobManager_RemoveEnqueue(t *testing.T) {
	id, err := testJobManager.Enqueue(context.Background(), "email", time.Now().Add(1*time.Minute))
	assert.NoError(t, err)
	err = testJobManager.RemoveEnqueue(context.Background(), id)
	assert.NoError(t, err)
}

func TestJobManager_FindEnqueue(t *testing.T) {
	id, err := testJobManager.Enqueue(context.Background(), "email", time.Now().Add(1*time.Minute))
	assert.NoError(t, err)

	job, err := testJobManager.FindEnqueue(context.Background(), id)
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, id, job.ID)
}

func TestJobManager_Schedule(t *testing.T) {
	id, err := testJobManager.Schedule(context.Background(), "daily", "*/2 * * * *", "foo", "bar")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}

func TestJobManager_ActivateAndDeactivateSchedule(t *testing.T) {
	id, err := testJobManager.Schedule(context.Background(), "daily", "*/3 * * * *", "x")
	assert.NoError(t, err)

	// No return value, just ensure it doesn't panic or error
	testJobManager.DeActivateSchedule(context.Background(), id)
	testJobManager.ActivateSchedule(context.Background(), id)
}

func TestJobManager_ScheduleEveryMinute(t *testing.T) {
	id, err := testJobManager.ScheduleEveryMinute(context.Background(), "heartbeat", "a")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}

func TestJobManager_ScheduleEveryHour(t *testing.T) {
	id, err := testJobManager.ScheduleEveryHour(context.Background(), "heartbeat", "b")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}

func TestJobManager_ScheduleEveryDay(t *testing.T) {
	id, err := testJobManager.ScheduleEveryDay(context.Background(), "heartbeat", "c")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}

func TestJobManager_ScheduleEveryWeek(t *testing.T) {
	id, err := testJobManager.ScheduleEveryWeek(context.Background(), "heartbeat", "d")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}

func TestJobManager_ScheduleEveryMonth(t *testing.T) {
	id, err := testJobManager.ScheduleEveryMonth(context.Background(), "heartbeat", "e")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}

func TestJobManager_ScheduleEveryYear(t *testing.T) {
	id, err := testJobManager.ScheduleEveryYear(context.Background(), "heartbeat", "f")
	assert.NoError(t, err)
	assert.True(t, id > 0)
}
