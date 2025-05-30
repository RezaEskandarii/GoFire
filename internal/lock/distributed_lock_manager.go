package lock

// DistributedLockManager defines the interface for acquiring and releasing distributed locks.
// Implementations should ensure that lock acquisition is safe across multiple instances of the application,
// typically backed by a shared resource like a database or a distributed cache.
//
// Acquire attempts to obtain a lock identified by lockID. It should return an error if the lock
// cannot be acquired (e.g. already held by another process).
//
// Release releases the lock identified by lockID. It should return an error if the lock is not held
// or cannot be released due to underlying issues.
type DistributedLockManager interface {
	Acquire(lockID int) error
	Release(lockID int) error
}
