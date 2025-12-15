package gofire

import (
	"database/sql"
	"fmt"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/message_broaker"
	"github.com/RezaEskandarii/gofire/internal/models/config"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/redis/go-redis/v9"
)

type JobManagers struct {
	EnqueuedJobStore store.EnqueuedJobStore
	CronJobStore     store.CronJobStore
	UserStore        store.UserStore
	LockMgr          lock.DistributedLockManager
	EnqueueScheduler enqueueJobsManager
	CronJobManager   cronJobManager
	MessageBroker    message_broaker.MessageBroker
}

// createJobManagers initializes all required job-related services and managers
// including job stores, distributed lock manager, schedulers, and optional message broker.
// ---------------------------------------------------------------------------------------------
func createJobManagers(cfg config.GofireConfig, sqlDB *sql.DB, redisClient *redis.Client, jobHandler *JobHandler) (*JobManagers, error) {

	// ---------------------------------------------------------------------------------------------
	// Initialize Job Stores (EnqueuedJobStore, CronJobStore, UserStore)
	// ---------------------------------------------------------------------------------------------
	enqueuedJobStore := CreateEnqueuedJobStore(cfg.StorageDriver, sqlDB, redisClient)
	cronJobStore := CreateCronJobStore(cfg.StorageDriver, sqlDB, redisClient)
	userStore := CreateUserStore(cfg.StorageDriver, sqlDB, redisClient)

	// ---------------------------------------------------------------------------------------------
	// Initialize Distributed Lock Manager (for controlling concurrency across multiple instances)
	// ---------------------------------------------------------------------------------------------
	lockMgr := CreateDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	// ---------------------------------------------------------------------------------------------
	// Initialize Cron Job Manager (manages scheduled jobs using cron definitions)
	// ---------------------------------------------------------------------------------------------
	cronJobManager := newCronJobManager(cronJobStore, lockMgr, jobHandler, cfg.Instance)

	// ---------------------------------------------------------------------------------------------
	// Initialize Message Broker (RabbitMQ), if queue writer is enabled
	// ---------------------------------------------------------------------------------------------
	rabbitCfg := cfg.RabbitMQConfig
	var messageBroker *message_broaker.RabbitMQ
	if cfg.UseQueueWriter {
		mBroker, err := message_broaker.NewRabbitMQ(rabbitCfg.URL, rabbitCfg.Exchange, rabbitCfg.Queue, "")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
		}
		messageBroker = mBroker
	}

	// ---------------------------------------------------------------------------------------------
	// Initialize Enqueue Scheduler (responsible for scheduling and dispatching enqueued jobs)
	// ---------------------------------------------------------------------------------------------
	enqueueScheduler := newEnqueueScheduler(enqueuedJobStore, lockMgr, jobHandler, messageBroker, cfg.Instance)

	// ---------------------------------------------------------------------------------------------
	// Assemble all components into JobManagers struct and return it
	// ---------------------------------------------------------------------------------------------
	return &JobManagers{
		EnqueuedJobStore: enqueuedJobStore,
		CronJobStore:     cronJobStore,
		UserStore:        userStore,
		LockMgr:          lockMgr,
		EnqueueScheduler: enqueueScheduler,
		CronJobManager:   cronJobManager,
		MessageBroker:    messageBroker,
	}, nil
}
