package di

import (
	"database/sql"
	"fmt"
	"github.com/RezaEskandarii/gofire/client"
	config2 "github.com/RezaEskandarii/gofire/types/config"
	"github.com/redis/go-redis/v9"
)

func GetDependencies(cfg *config2.GofireConfig) (*config2.JobHandler, *JobDependency, *client.JobManager, error) {

	sqlDB, redisClient, err := getStorageConnections(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	jobHandler := config2.NewJobHandler()

	dependencies, err := createJobDependency(cfg, sqlDB, redisClient, jobHandler)
	if err != nil {
		return nil, nil, nil, err
	}

	jm := client.NewJobManager(
		dependencies.EnqueuedJobStore,
		dependencies.CronJobStore,
		jobHandler,
		dependencies.LockMgr,
		dependencies.MessageBroker,
		cfg.UseQueueWriter,
		cfg.RabbitMQConfig.Queue,
	)

	return jobHandler, dependencies, jm, err
}

// getStorageConnections sets up storage backends (Postgres or Redis) based on the configuration.
// Returns the initialized SQL DB, Redis client, and an error if the driver is unsupported.
func getStorageConnections(cfg *config2.GofireConfig) (*sql.DB, *redis.Client, error) {
	var sqlDB *sql.DB
	var redisClient *redis.Client

	switch cfg.StorageDriver {
	case config2.Postgres:
		sqlDB = getPG(cfg.PostgresConfig.ConnectionUrl)
		//setPostgresConnectionPool(sqlDB)

	//case config.Redis:
	//	panic("redis storage driver not yet supported")
	default:
		return nil, nil, fmt.Errorf("unsupported driver: %v", cfg.StorageDriver)
	}
	return sqlDB, redisClient, nil
}
