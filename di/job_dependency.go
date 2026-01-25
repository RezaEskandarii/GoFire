package di

import (
	"database/sql"
	"fmt"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/types/config"

	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/message_broaker"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/redis/go-redis/v9"
)

type JobDependency struct {
	EnqueuedJobStore store.EnqueuedJobStore
	CronJobStore     store.CronJobStore
	UserStore        store.UserStore
	LockMgr          lock.DistributedLockManager
	EnqueueScheduler client.EnqueueJobsManager
	CronJobManager   client.CronJobManager
	MessageBroker    message_broaker.MessageBroker
}

// createJobDependency initializes all required job-related services and managers
// including job stores, distributed lock manager, schedulers, and optional message broker.
// ---------------------------------------------------------------------------------------------
func createJobDependency(cfg *config.GofireConfig, sqlDB *sql.DB, redisClient *redis.Client, jobHandler *config.JobHandler) (*JobDependency, error) {

	enqueuedJobStore := createEnqueuedJobStore(cfg.StorageDriver, sqlDB, redisClient)
	cronJobStore := createCronJobStore(cfg.StorageDriver, sqlDB, redisClient)
	userStore := createUserStore(cfg.StorageDriver, sqlDB, redisClient)

	lockMgr := createDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	cronJobManager := client.NewCronJobManager(cronJobStore, lockMgr, jobHandler, cfg.Instance)

	rabbitCfg := cfg.RabbitMQConfig
	var messageBroker *message_broaker.RabbitMQ
	if cfg.UseQueueWriter {
		mBroker, err := message_broaker.NewRabbitMQ(rabbitCfg.URL, rabbitCfg.Exchange, rabbitCfg.Queue, "")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
		}
		messageBroker = mBroker
	}

	enqueueScheduler := client.NewEnqueueScheduler(enqueuedJobStore, lockMgr, jobHandler, messageBroker, cfg.Instance)

	return &JobDependency{
		EnqueuedJobStore: enqueuedJobStore,
		CronJobStore:     cronJobStore,
		UserStore:        userStore,
		LockMgr:          lockMgr,
		EnqueueScheduler: enqueueScheduler,
		CronJobManager:   cronJobManager,
		MessageBroker:    messageBroker,
	}, nil
}
