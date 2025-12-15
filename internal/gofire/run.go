package gofire

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/RezaEskandarii/gofire/internal/db"
	"github.com/RezaEskandarii/gofire/internal/models/config"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/web"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"log"
	"runtime"
)

// BootJobManager initializes the entire Gofire job scheduling and execution system using the provided GofireConfig.
//
// It dynamically sets up storage backends (PostgreSQL or Redis), registers job handlers,
// initializes the Storesitories and distributed lock manager, and launches all necessary background services,
// including job enqueuing, cron evaluation, and optionally the admin dashboard.
//
// The function performs the following steps:
//  1. Connects to the specified storage backend based on cfg.StorageDriver (Postgres or Redis).
//  2. Registers all job handlers defined in cfg.Handlers.
//  3. Instantiates Storesitories, schedulers, and locking mechanisms using createJobManagers.
//  4. Runs schema and migration setup if PostgreSQL is used (protected by a distributed lock).
//  5. Starts background workers for scheduled and recurring jobs.
//  6. Launches a web dashboard for job monitoring and manual control (if enabled).
//
// Parameters:
//   - ctx: context used for cancellation and timeout propagation to background workers.
//   - cfg: full configuration of the system (including storage, handlers, worker count, dashboard options).
//
// Returns:
//   - JobManager: a configured job manager ready to enqueue, execute, and monitor jobs.
//   - error: any failure that prevents full system setup (e.g., invalid config, failed connection, migration error).
func BootJobManager(ctx context.Context, cfg config.GofireConfig) (*JobManager, error) {

	log.Printf("GOMAXPROCS Is: %d\n", runtime.GOMAXPROCS(0))
	// ---------------------------------------------------------------------------------------------
	// Recover from any panic and log the error (for safety in booting process)
	// ---------------------------------------------------------------------------------------------
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	// ---------------------------------------------------------------------------------------------
	// Initialize storage connections (PostgreSQL)
	// ---------------------------------------------------------------------------------------------
	sqlDB, redisClient, err := getStorageConnections(cfg)
	if err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------------------------
	// Create and register all job handlers as defined in configuration
	// ---------------------------------------------------------------------------------------------
	jobHandler := NewJobHandler()
	for _, handler := range cfg.Handlers {
		if err := jobHandler.Register(handler.JobName, handler.Func); err != nil {
			return nil, err
		}
	}

	// ---------------------------------------------------------------------------------------------
	// Create all job-related managers (stores, lock manager, schedulers, message broker, etc.)
	// ---------------------------------------------------------------------------------------------
	managers, err := createJobManagers(cfg, sqlDB, redisClient, jobHandler)
	if err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------------------------
	// Initialize database with lock manager (for schema setup or other bootstrapping logic)
	// ---------------------------------------------------------------------------------------------
	if err = db.Init(cfg.PostgresConfig.ConnectionUrl, managers.LockMgr); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------------------------
	// If admin user is defined in config, create the dashboard admin account
	// ---------------------------------------------------------------------------------------------
	if err = createDashboardAdminIfConfigured(ctx, &cfg, managers.UserStore); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------------------------
	// Start dashboard server if authentication is enabled
	// ---------------------------------------------------------------------------------------------
	if cfg.DashboardAuthEnabled {
		runServer(managers, cfg)
	}

	// ---------------------------------------------------------------------------------------------
	// Create the main JobManager with initialized components
	// ---------------------------------------------------------------------------------------------
	jm := NewJobManager(
		managers.EnqueuedJobStore,
		managers.CronJobStore,
		jobHandler,
		managers.LockMgr,
		managers.MessageBroker,
		cfg.UseQueueWriter,
		cfg.RabbitMQConfig.Queue,
	)

	// ---------------------------------------------------------------------------------------------
	// Prepare context and wait group for job reader goroutines
	// ---------------------------------------------------------------------------------------------
	jobCtx, cancel := context.WithCancel(ctx)
	jm.cancel = cancel
	jm.wg.Add(2) // Add the number of job reader goroutines to wait for

	// ---------------------------------------------------------------------------------------------
	// Start goroutines for job readers (e.g., cron reader, enqueued job reader)
	// ---------------------------------------------------------------------------------------------
	startJobReaders(jobCtx, jm, managers, cfg)

	// ---------------------------------------------------------------------------------------------
	// Return the fully initialized JobManager
	// ---------------------------------------------------------------------------------------------
	return jm, nil
}

// runServer initializes and starts the web server for the dashboard interface in a separate goroutine.
func runServer(managers *JobManagers, cfg config.GofireConfig) {
	go func() {
		router := web.NewRouteHandler(managers.EnqueuedJobStore, managers.UserStore, managers.CronJobStore, cfg.SecretKey, cfg.DashboardAuthEnabled, cfg.DashboardPort)
		if err := router.Serve(); err != nil {
			log.Printf("failed to start server: %v", err)
		}
	}()
}

// getStorageConnections sets up storage backends (Postgres or Redis) based on the configuration.
// Returns the initialized SQL DB, Redis client, and an error if the driver is unsupported.
func getStorageConnections(cfg config.GofireConfig) (*sql.DB, *redis.Client, error) {
	var sqlDB *sql.DB
	var redisClient *redis.Client

	switch cfg.StorageDriver {
	case config.Postgres:
		sqlDB = setupPostgres(cfg.PostgresConfig.ConnectionUrl)
		setPostgresConnectionPool(sqlDB)

	case config.Redis:
		panic("redis storage driver not yet supported")
	default:
		return nil, nil, fmt.Errorf("unsupported driver: %v", cfg.StorageDriver)
	}
	return sqlDB, redisClient, nil
}

// startJobReaders launches goroutines for processing enqueued and cron jobs.
// Uses the JobManager wait group to track job reader lifecycle.
func startJobReaders(ctx context.Context, jm *JobManager, managers *JobManagers, cfg config.GofireConfig) {

	// ---------------------------------------------------------------------------------------------
	// Start Enqueue Scheduler in a separate goroutine
	// Responsible for processing enqueued jobs periodically.
	// ---------------------------------------------------------------------------------------------
	go func() {
		defer jm.wg.Done()

		go func() {
			if err := managers.EnqueueScheduler.Start(ctx, cfg.EnqueueInterval, cfg.WorkerCount, cfg.BatchSize); err != nil {
				log.Printf("EnqueueScheduler failed to start: %v", err)
			}
		}()
	}()

	// ---------------------------------------------------------------------------------------------
	// Start Cron Job Manager in a separate goroutine
	// Responsible for handling cron-based scheduled jobs.
	// ---------------------------------------------------------------------------------------------
	go func() {
		defer jm.wg.Done()

		go func() {
			if err := managers.CronJobManager.Start(ctx, cfg.ScheduleInterval, cfg.WorkerCount, cfg.BatchSize); err != nil {
				log.Printf("CronJobManager failed to start: %v", err)
			}
		}()
	}()

	// ---------------------------------------------------------------------------------------------
	// Start Queue and Storage Sync Worker if queue writer is enabled
	// Syncs storage with message broker (RabbitMQ).
	// ---------------------------------------------------------------------------------------------
	if cfg.UseQueueWriter {
		go func() {
			if err := managers.EnqueueScheduler.StartQueueAndStorageSyncWorker(ctx, cfg.RabbitMQConfig.Queue, cfg.UseQueueWriter); err != nil {
				log.Printf("EnqueueScheduler failed to startQueueAndStorageSyncWorker: %v", err)
			}
		}()
	}
}

// setPostgresConnectionPool configures the Postgres connection pool with recommended limits.
func setPostgresConnectionPool(sqlDB *sql.DB) {
	sqlDB.SetMaxOpenConns(80)
	sqlDB.SetMaxIdleConns(30)
}

// createDashboardAdminIfConfigured creates the default dashboard user if credentials are provided
// and the user does not already exist in the store.
func createDashboardAdminIfConfigured(ctx context.Context, cfg *config.GofireConfig, userStore store.UserStore) error {
	if cfg.DashboardUserName != "" && cfg.DashboardPassword != "" {
		if user, err := userStore.FindByUsername(ctx, cfg.DashboardUserName); err == nil && user == nil {

			if _, err := userStore.Create(ctx, cfg.DashboardUserName, cfg.DashboardPassword); err != nil {
				return err
			}

		} else {
			return err
		}
	}
	return nil
}
