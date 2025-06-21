package test

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gofire/internal/lock"
	"testing"
)

func TestPostgresDistributedLockManager_Acquire_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	lockID := 123
	mock.ExpectExec("SELECT pg_advisory_lock\\(\\$1\\)").
		WithArgs(lockID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	manager := lock.NewPostgresDistributedLockManager(db)
	err = manager.Acquire(lockID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDistributedLockManager_Acquire_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	lockID := 123
	mock.ExpectExec("SELECT pg_advisory_lock\\(\\$1\\)").
		WithArgs(lockID).
		WillReturnError(assert.AnError)

	manager := lock.NewPostgresDistributedLockManager(db)
	err = manager.Acquire(lockID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acquire lock")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDistributedLockManager_Release_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	lockID := 123
	mock.ExpectExec("SELECT pg_advisory_unlock\\(\\$1\\)").
		WithArgs(lockID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	manager := lock.NewPostgresDistributedLockManager(db)
	err = manager.Release(lockID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresDistributedLockManager_Release_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	lockID := 123
	mock.ExpectExec("SELECT pg_advisory_unlock\\(\\$1\\)").
		WithArgs(lockID).
		WillReturnError(assert.AnError)

	manager := lock.NewPostgresDistributedLockManager(db)
	err = manager.Release(lockID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to release lock")
	assert.NoError(t, mock.ExpectationsWereMet())
}
