package gofire

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
	"gofire/internal/models/config"
	"gofire/internal/store"
	"gofire/internal/store/postgres"
)

func CreateEnqueuedJobStore(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) store.EnqueuedJobStore {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresEnqueuedJobStore(db)
	case config.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateCronJobStore(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) store.CronJobStore {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresCronJobStore(db)
	case config.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateUserStore(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) store.UserStore {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresUserStore(db)
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
