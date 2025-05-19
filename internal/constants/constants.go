package constants

const (
	MigrationLock = iota
	EnqueueLock
	ScheduleLock
	RetryLock
)

const (
	MaxRetryAttempt = 3
)
