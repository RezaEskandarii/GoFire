package config

import (
	"database/sql"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/internal/store/postgres"
	"github.com/redis/go-redis/v9"
)

func CreateEnqueuedJobStore(driver StorageDriver, db *sql.DB, redisClient *redis.Client) store.EnqueuedJobStore {
	switch driver {
	case Postgres:
		return postgres.NewPostgresEnqueuedJobStore(db)
	case Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateCronJobStore(driver StorageDriver, db *sql.DB, redisClient *redis.Client) store.CronJobStore {
	switch driver {
	case Postgres:
		return postgres.NewPostgresCronJobStore(db)
	case Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateUserStore(driver StorageDriver, db *sql.DB, redisClient *redis.Client) store.UserStore {
	switch driver {
	case Postgres:
		return postgres.NewPostgresUserStore(db)
	case Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func CreateDistributedLockManager(driver StorageDriver, db *sql.DB, redisClient *redis.Client) lock.DistributedLockManager {
	switch driver {
	case Postgres:
		return lock.NewPostgresDistributedLockManager(db)
	case Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}
