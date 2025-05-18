package lock

type DistributedLockManager interface {
	Acquire(lockID int) error
	Release(lockID int) error
}
