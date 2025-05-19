package gofire

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
	"gofire/internal/lock"
	"gofire/internal/models"
	"gofire/internal/repository"
)

func NewEnqueuedJobRepository(driver models.StorageDriver, db *sql.DB, redisClient *redis.Client) repository.EnqueuedJobRepository {
	switch driver {
	case models.Postgres:
		return repository.NewPostgresEnqueuedJobRepository(db)
	case models.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}

func NewDistributedLockManager(driver models.StorageDriver, db *sql.DB, redisClient *redis.Client) lock.DistributedLockManager {
	switch driver {
	case models.Postgres:
		return lock.NewPostgresDistributedLockManager(db)
	case models.Redis:
		panic("unsupported storage driver")
	default:
		panic("unsupported storage driver")
	}
}
