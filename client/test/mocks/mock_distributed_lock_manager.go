package mocks

// MockDistributedLockManager is a mock implementation of lock.DistributedLockManager for testing.
type MockDistributedLockManager struct {
	AcquireFunc func(lockID int) error
	ReleaseFunc func(lockID int) error
}

func (m *MockDistributedLockManager) Acquire(lockID int) error {
	if m.AcquireFunc != nil {
		return m.AcquireFunc(lockID)
	}
	return nil
}

func (m *MockDistributedLockManager) Release(lockID int) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(lockID)
	}
	return nil
}
