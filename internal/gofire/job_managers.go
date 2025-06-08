package gofire

import (
	"database/sql"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
	"gofire/internal/message_broaker"
	"gofire/internal/models/config"
	"gofire/internal/repository"
)

type JobManagers struct {
	EnqueuedJobRepo  repository.EnqueuedJobRepository
	CronJobRepo      repository.CronJobRepository
	UserRepo         repository.UserRepository
	LockMgr          lock.DistributedLockManager
	EnqueueScheduler enqueueJobsManager
	CronJobManager   cronJobManager
	MessageBroker    message_broaker.MessageBroker
}

func createJobManagers(cfg config.GofireConfig, sqlDB *sql.DB, redisClient *redis.Client, jobHandler JobHandler) (*JobManagers, error) {
	enqueuedJobRepo := CreateEnqueuedJobRepository(cfg.StorageDriver, sqlDB, redisClient)
	cronJobRepo := CreateCronJobRepository(cfg.StorageDriver, sqlDB, redisClient)
	userRepo := CreateUserRepository(cfg.StorageDriver, sqlDB, redisClient)

	lockMgr := CreateDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	cronJobManager := newCronJobManager(cronJobRepo, lockMgr, jobHandler, cfg.Instance)

	rabbitCfg := cfg.RabbitMQConfig

	var messageBroker *message_broaker.RabbitMQ
	if cfg.UseQueueWriter {
		mBroker, err := message_broaker.NewRabbitMQ(rabbitCfg.URL, rabbitCfg.Exchange, rabbitCfg.Queue, "")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
		}
		messageBroker = mBroker
	}

	enqueueScheduler := newEnqueueScheduler(enqueuedJobRepo, lockMgr, jobHandler, messageBroker, cfg.Instance)
	return &JobManagers{
		EnqueuedJobRepo:  enqueuedJobRepo,
		CronJobRepo:      cronJobRepo,
		UserRepo:         userRepo,
		LockMgr:          lockMgr,
		EnqueueScheduler: enqueueScheduler,
		CronJobManager:   cronJobManager,
		MessageBroker:    messageBroker,
	}, nil
}
