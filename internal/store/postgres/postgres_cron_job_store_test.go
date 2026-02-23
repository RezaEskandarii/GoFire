package postgres

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewPostgresCronJobStore(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	require.NotNil(t, store)
}

func TestPostgresCronJobStore_AddOrUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()
	scheduledAt := time.Now().Add(time.Hour)

	mock.ExpectQuery("INSERT INTO gofire_schema.cron_jobs").
		WithArgs("CronJob", scheduledAt, sqlmock.AnyArg(), "* * * * *", state.StatusQueued).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	jobID, err := store.AddOrUpdate(ctx, "CronJob", scheduledAt, "* * * * *", "arg1")
	require.NoError(t, err)
	assert.Equal(t, int64(7), jobID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_AddOrUpdate_InvalidPayload(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	_, err = store.AddOrUpdate(ctx, "Job", time.Now(), "* * * * *", make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal payload")
}

func TestPostgresCronJobStore_LockJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WithArgs("instance-1", state.StatusProcessing, 1, state.StatusQueued, state.StatusRetrying, state.StatusSucceeded, state.StatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := store.LockJob(ctx, 1, "instance-1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_LockJob_NotAcquired(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))

	ok, err := store.LockJob(ctx, 1, "instance-1")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_MarkSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WithArgs(state.StatusSucceeded, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.MarkSuccess(ctx, 1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_MarkFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WithArgs("error msg", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.MarkFailure(ctx, 1, "error msg")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_Activate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WithArgs(true, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.Activate(ctx, 1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_DeActivate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WithArgs(false, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.DeActivate(ctx, 1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_UpdateJobRunTimes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresCronJobStore(db)
	ctx := context.Background()
	lastRun := time.Now()
	nextRun := time.Now().Add(time.Hour)

	mock.ExpectExec("UPDATE gofire_schema.cron_jobs").
		WithArgs(lastRun, nextRun, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.UpdateJobRunTimes(ctx, 1, lastRun, nextRun)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCronJobStore_Close(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	mock.ExpectClose()

	store := NewPostgresCronJobStore(db)
	err = store.Close()
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
