package db

import (
	"errors"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type mockLockManager struct {
	acquireErr error
	releaseErr error
}

func (m *mockLockManager) Acquire(lockID int) error { return m.acquireErr }
func (m *mockLockManager) Release(lockID int) error { return m.releaseErr }

func TestReadSQLScripts(t *testing.T) {
	scripts, err := readSQLScripts()
	require.NoError(t, err)
	assert.NotEmpty(t, scripts)
	assert.GreaterOrEqual(t, len(scripts), 4) // cron_jobs, enqueued_jobs, schema_state, schema_users
}

func TestInit_LockAcquireFails(t *testing.T) {
	lockMgr := &mockLockManager{acquireErr: errors.New("lock busy")}

	err := Init("postgres://invalid", lockMgr)
	assert.Error(t, err)
}

func TestInit_LockAcquireSucceedsButPingFails(t *testing.T) {
	lockMgr := &mockLockManager{}

	// Invalid connection URL - Ping will fail
	err := Init("postgres://user:pass@invalidhost:9999/nonexistent?sslmode=disable", lockMgr)
	assert.Error(t, err)
}

var _ lock.DistributedLockManager = (*mockLockManager)(nil)
