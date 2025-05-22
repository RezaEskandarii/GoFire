package lock

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=boofhichkas dbname=esim sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	return db
}

func TestNewPostgresDistributedLockManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewPostgresDistributedLockManager(db)
	if manager == nil {
		t.Error("NewPostgresDistributedLockManager() returned nil")
	}
	if manager.db != db {
		t.Error("NewPostgresDistributedLockManager() did not set the database connection")
	}
}

func TestPostgresDistributedLockManager_Acquire(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewPostgresDistributedLockManager(db)
	lockID := 1

	// Test successful lock acquisition
	err := manager.Acquire(lockID)
	if err != nil {
		t.Errorf("Acquire() error = %v", err)
	}

	// Test lock release
	err = manager.Release(lockID)
	if err != nil {
		t.Errorf("Release() error = %v", err)
	}

	// Test concurrent lock acquisition
	done := make(chan bool)
	go func() {
		err := manager.Acquire(lockID)
		if err != nil {
			t.Errorf("Concurrent Acquire() error = %v", err)
		}
		time.Sleep(100 * time.Millisecond)
		manager.Release(lockID)
		done <- true
	}()

	// Try to acquire the same lock
	err = manager.Acquire(lockID)
	if err != nil {
		t.Errorf("Second Acquire() error = %v", err)
	}
	manager.Release(lockID)

	<-done
}

func TestPostgresDistributedLockManager_Release(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewPostgresDistributedLockManager(db)
	lockID := 2

	// Test releasing a lock that hasn't been acquired
	err := manager.Release(lockID)
	if err != nil {
		t.Errorf("Release() error = %v", err)
	}

	// Test releasing an acquired lock
	err = manager.Acquire(lockID)
	if err != nil {
		t.Errorf("Acquire() error = %v", err)
	}

	err = manager.Release(lockID)
	if err != nil {
		t.Errorf("Release() error = %v", err)
	}

	// Test releasing the same lock multiple times
	err = manager.Release(lockID)
	if err != nil {
		t.Errorf("Second Release() error = %v", err)
	}
}
