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

var Locks = []int{
	MigrationLock,
	EnqueueLock,
	StartEnqueueLock,
	ScheduleLock,
	RetryLock,
	CronJobLock,
	StartCronJobLock,
}

const (
	MaxRetryAttempt = 3
)
