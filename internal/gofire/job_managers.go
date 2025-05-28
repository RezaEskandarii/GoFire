package gofire

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
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
}

func createJobManagers(cfg config.GofireConfig, sqlDB *sql.DB, redisClient *redis.Client, jobHandler JobHandler) (*JobManagers, error) {
	enqueuedJobRepo := CreateEnqueuedJobRepository(cfg.StorageDriver, sqlDB, redisClient)
	cronJobRepo := CreateCronJobRepository(cfg.StorageDriver, sqlDB, redisClient)
	userRepo := CreateUserRepository(cfg.StorageDriver, sqlDB, redisClient)

	lockMgr := CreateDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	enqueueScheduler := newEnqueueScheduler(enqueuedJobRepo, lockMgr, jobHandler, cfg.Instance)
	cronJobManager := newCronJobManager(cronJobRepo, lockMgr, jobHandler, cfg.Instance)

	return &JobManagers{
		EnqueuedJobRepo:  enqueuedJobRepo,
		CronJobRepo:      cronJobRepo,
		UserRepo:         userRepo,
		LockMgr:          lockMgr,
		EnqueueScheduler: enqueueScheduler,
		CronJobManager:   cronJobManager,
	}, nil
}
