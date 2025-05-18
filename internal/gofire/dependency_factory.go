package gofire

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
	"gofire/internal/repository"
)

func NewEnqueuedJobRepository(driver StorageDriver, db *sql.DB, redisClient *redis.Client) repository.EnqueuedJobRepository {
	switch driver {
	case Postgres:
		return repository.NewPostgresEnqueuedJobRepository(db)
	case Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func NewDistributedLockManager(driver StorageDriver, db *sql.DB, redisClient *redis.Client) lock.DistributedLockManager {
	switch driver {
	case Postgres:
		return lock.NewPostgresDistributedLockManager(db)
	case Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}
