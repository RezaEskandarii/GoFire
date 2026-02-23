package postgres

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RezaEskandarii/gofire/internal/state"
	"github.com/RezaEskandarii/gofire/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewPostgresEnqueuedJobStore(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	require.NotNil(t, store)
}

func TestPostgresEnqueuedJobStore_Insert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()
	scheduledAt := time.Now().Add(time.Hour)

	mock.ExpectQuery("INSERT INTO gofire_schema.enqueued_jobs").
		WithArgs("TestJob", sqlmock.AnyArg(), scheduledAt, 3).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	jobID, err := store.Insert(ctx, "TestJob", scheduledAt, "arg1", "arg2")
	require.NoError(t, err)
	assert.Equal(t, int64(42), jobID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_Insert_MarshalError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	// Invalid JSON - channel can't be marshaled
	_, err = store.Insert(ctx, "Job", time.Now(), make(chan int))
	assert.Error(t, err)
}

func TestPostgresEnqueuedJobStore_RemoveByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM gofire_schema.enqueued_jobs").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.RemoveByID(ctx, 1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_RemoveByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM gofire_schema.enqueued_jobs").
		WithArgs(999).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.RemoveByID(ctx, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no job found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_LockJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.enqueued_jobs").
		WithArgs("instance-1", state.StatusProcessing, 1, state.StatusQueued, state.StatusRetrying).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := store.LockJob(ctx, 1, "instance-1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_LockJob_NotAcquired(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.enqueued_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))

	ok, err := store.LockJob(ctx, 1, "instance-1")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_MarkSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.enqueued_jobs").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.MarkSuccess(ctx, 1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_MarkFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectExec("UPDATE gofire_schema.enqueued_jobs").
		WithArgs(1, "error msg", state.StatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.MarkFailure(ctx, 1, "error msg", 1, 3)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_CountJobsByStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(state.StatusQueued).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	count, err := store.CountJobsByStatus(ctx, state.StatusQueued)
	require.NoError(t, err)
	assert.Equal(t, 10, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresEnqueuedJobStore_BulkInsert_Empty(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()

	// BulkInsert with empty batch is a no-op
	err = store.BulkInsert(ctx, nil)
	require.NoError(t, err)

	err = store.BulkInsert(ctx, []types.Job{})
	require.NoError(t, err)
}

func TestPostgresEnqueuedJobStore_BulkInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewPostgresEnqueuedJobStore(db)
	ctx := context.Background()
	now := time.Now()

	batch := []types.Job{
		{Name: "Job1", Args: []any{"a"}, ScheduledAt: now},
		{Name: "Job2", Args: []any{"b"}, ScheduledAt: now},
	}

	mock.ExpectExec("INSERT INTO gofire_schema.enqueued_jobs").
		WillReturnResult(sqlmock.NewResult(0, 2))

	err = store.BulkInsert(ctx, batch)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
