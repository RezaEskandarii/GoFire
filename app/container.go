package app

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/internal/lock"
	"github.com/RezaEskandarii/gofire/internal/message_broaker"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/types/config"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Container holds all application dependencies. It is the single source of truth
// for dependency injection and ensures connections and services are created once.
type Container struct {
	Config *config.GofireConfig

	// Storage connections (created once, shared by all stores)
	DB    *sql.DB
	Redis *redis.Client

	// Stores (implement interfaces for testability)
	EnqueuedJobStore store.EnqueuedJobStore
	CronJobStore     store.CronJobStore
	UserStore        store.UserStore

	// Infrastructure
	LockManager   lock.DistributedLockManager
	MessageBroker message_broaker.MessageBroker

	// Job handlers and managers
	JobHandler       *config.JobHandler
	EnqueueScheduler client.EnqueueJobsManager
	CronJobManager   client.CronJobManager
	JobManager       *client.JobManager
}

// NewContainer creates and wires all dependencies. Single entry point for DI.
// Call this once per application lifecycle.
// Pass optional WithDB, WithRedis to inject connections for testing.
func NewContainer(ctx context.Context, cfg *config.GofireConfig, opts ...ContainerOption) (*Container, error) {
	opt := &containerConfig{}
	for _, o := range opts {
		o(opt)
	}

	var db *sql.DB
	var redisClient *redis.Client
	var err error

	if opt.db != nil {
		db = opt.db
		redisClient = opt.redis
	} else {
		db, redisClient, err = initStorageConnections(cfg)
		if err != nil {
			return nil, fmt.Errorf("init storage: %w", err)
		}
	}

	jobHandler := config.NewJobHandler()

	enqueuedStore := createEnqueuedJobStore(cfg.StorageDriver, db, redisClient)
	cronStore := createCronJobStore(cfg.StorageDriver, db, redisClient)
	userStore := createUserStore(cfg.StorageDriver, db, redisClient)
	lockMgr := createDistributedLockManager(cfg.StorageDriver, db, redisClient)

	cronJobManager := client.NewCronJobManager(cronStore, lockMgr, jobHandler, cfg.Instance)

	var messageBroker message_broaker.MessageBroker
	if cfg.UseQueueWriter {
		mBroker, err := message_broaker.NewRabbitMQ(
			cfg.RabbitMQConfig.URL,
			cfg.RabbitMQConfig.Exchange,
			cfg.RabbitMQConfig.Queue,
			"",
		)
		if err != nil {
			return nil, fmt.Errorf("init rabbitmq: %w", err)
		}
		messageBroker = mBroker
	}

	enqueueScheduler := client.NewEnqueueScheduler(enqueuedStore, lockMgr, jobHandler, messageBroker, cfg.Instance)

	jobManager := client.NewJobManager(
		enqueuedStore,
		cronStore,
		jobHandler,
		lockMgr,
		messageBroker,
		cfg.UseQueueWriter,
		cfg.RabbitMQConfig.Queue,
	)

	return &Container{
		Config:           cfg,
		DB:               db,
		Redis:            redisClient,
		EnqueuedJobStore: enqueuedStore,
		CronJobStore:     cronStore,
		UserStore:        userStore,
		LockManager:      lockMgr,
		MessageBroker:    messageBroker,
		JobHandler:       jobHandler,
		EnqueueScheduler: enqueueScheduler,
		CronJobManager:   cronJobManager,
		JobManager:       jobManager,
	}, nil
}

// initStorageConnections creates database connections based on config.
func initStorageConnections(cfg *config.GofireConfig) (*sql.DB, *redis.Client, error) {
	switch cfg.StorageDriver {
	case config.Postgres:
		db, err := openPostgresDB(cfg.PostgresConfig.ConnectionUrl)
		if err != nil {
			return nil, nil, err
		}
		return db, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported storage driver: %v", cfg.StorageDriver)
	}
}

func openPostgresDB(connectionURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	// Optional: configure connection pool
	// db.SetMaxOpenConns(25)
	// db.SetMaxIdleConns(5)
	return db, nil
}
