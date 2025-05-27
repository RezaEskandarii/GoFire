package constants

const (
	MigrationLock = iota
	EnqueueLock
	StartEnqueueLock
	ScheduleLock
	RetryLock
	CronJobLock
	StartCronJobLock
)

const (
	MaxRetryAttempt = 3
)
