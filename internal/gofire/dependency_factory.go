package gofire

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
	"gofire/internal/models/config"
	"gofire/internal/repository"
	"gofire/internal/repository/postgres"
)

func CreateEnqueuedJobRepository(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) repository.EnqueuedJobRepository {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresEnqueuedJobRepository(db)
	case config.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateCronJobRepository(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) repository.CronJobRepository {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresCronJobRepository(db)
	case config.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateDistributedLockManager(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) lock.DistributedLockManager {
	switch driver {
	case config.Postgres:
		return lock.NewPostgresDistributedLockManager(db)
	case config.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}
