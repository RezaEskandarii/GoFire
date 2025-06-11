package mocks

import (
	"errors"
	"sync"
)

type MockDistributedLockManager struct {
	mu    sync.Mutex
	locks map[int]bool
}

func NewMockDistributedLockManager() *MockDistributedLockManager {
	return &MockDistributedLockManager{
		locks: make(map[int]bool),
	}
}

func (m *MockDistributedLockManager) Acquire(lockID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.locks[lockID] {
		return errors.New("lock already acquired")
	}

	m.locks[lockID] = true
	return nil
}

func (m *MockDistributedLockManager) Release(lockID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.locks[lockID] {
		return errors.New("lock not held")
	}

	delete(m.locks, lockID)
	return nil
}
