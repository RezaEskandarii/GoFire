package gofire

import (
	"database/sql"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
	"gofire/internal/message_broaker"
	"gofire/internal/models/config"
	"gofire/internal/store"
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

func createJobManagers(cfg config.GofireConfig, sqlDB *sql.DB, redisClient *redis.Client, jobHandler *JobHandler) (*JobManagers, error) {
	enqueuedJobStore := CreateEnqueuedJobStore(cfg.StorageDriver, sqlDB, redisClient)
	cronJobStore := CreateCronJobStore(cfg.StorageDriver, sqlDB, redisClient)
	userStore := CreateUserStore(cfg.StorageDriver, sqlDB, redisClient)

	lockMgr := CreateDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	cronJobManager := newCronJobManager(cronJobStore, lockMgr, jobHandler, cfg.Instance)

	rabbitCfg := cfg.RabbitMQConfig

	var messageBroker *message_broaker.RabbitMQ
	if cfg.UseQueueWriter {
		mBroker, err := message_broaker.NewRabbitMQ(rabbitCfg.URL, rabbitCfg.Exchange, rabbitCfg.Queue, "")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
		}
		messageBroker = mBroker
	}

	enqueueScheduler := newEnqueueScheduler(enqueuedJobStore, lockMgr, jobHandler, messageBroker, cfg.Instance)
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
