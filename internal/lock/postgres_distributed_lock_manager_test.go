package lock

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPostgresDistributedLockManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mgr := NewPostgresDistributedLockManager(db)
	require.NotNil(t, mgr)
}

func TestPostgresDistributedLockManager_Acquire(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mgr := NewPostgresDistributedLockManager(db)

	mock.ExpectExec("SELECT pg_advisory_lock").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = mgr.Acquire(1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDistributedLockManager_Acquire_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mgr := NewPostgresDistributedLockManager(db)

	mock.ExpectExec("SELECT pg_advisory_lock").
		WithArgs(42).
		WillReturnError(sql.ErrConnDone)

	err = mgr.Acquire(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acquire lock")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDistributedLockManager_Release(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mgr := NewPostgresDistributedLockManager(db)

	mock.ExpectExec("SELECT pg_advisory_unlock").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = mgr.Release(1)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDistributedLockManager_Release_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mgr := NewPostgresDistributedLockManager(db)

	mock.ExpectExec("SELECT pg_advisory_unlock").
		WithArgs(99).
		WillReturnError(sql.ErrConnDone)

	err = mgr.Release(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to release lock")
	assert.NoError(t, mock.ExpectationsWereMet())
}
