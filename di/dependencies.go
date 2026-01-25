package di

import (
	"database/sql"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/internal/store/postgres"
	"github.com/RezaEskandarii/gofire/types/config"
	"github.com/redis/go-redis/v9"
)

func createEnqueuedJobStore(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) store.EnqueuedJobStore {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresEnqueuedJobStore(db)
	//case config.Redis:
	//	panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func createCronJobStore(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) store.CronJobStore {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresCronJobStore(db)
	//case config.Redis:
	//	panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func createUserStore(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) store.UserStore {
	switch driver {
	case config.Postgres:
		return postgres.NewPostgresUserStore(db)
	//case config.Redis:
	//	panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func createDistributedLockManager(driver config.StorageDriver, db *sql.DB, redisClient *redis.Client) lock.DistributedLockManager {
	switch driver {
	case config.Postgres:
		return lock.NewPostgresDistributedLockManager(db)
	//case config.Redis:
	//	panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}
