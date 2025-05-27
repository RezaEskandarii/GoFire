package constants

const (
	MigrationLock = iota
	EnqueueLock
	ScheduleLock
	RetryLock
	CronJobLock
)

const (
	MaxRetryAttempt = 3
)
