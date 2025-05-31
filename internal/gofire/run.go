package gofire

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gofire/internal/db"
	"gofire/internal/models/config"
	"gofire/internal/repository"
	"gofire/web"
	"log"
)

// SetUp initializes the entire Gofire job scheduling and execution system using the provided GofireConfig.
//
// It dynamically sets up storage backends (PostgreSQL or Redis), registers job handlers,
// initializes the repositories and distributed lock manager, and launches all necessary background services,
// including job enqueuing, cron evaluation, and optionally the admin dashboard.
//
// The function performs the following steps:
//  1. Connects to the specified storage backend based on cfg.StorageDriver (Postgres or Redis).
//  2. Registers all job handlers defined in cfg.Handlers.
//  3. Instantiates repositories, schedulers, and locking mechanisms using createJobManagers.
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
func SetUp(ctx context.Context, cfg config.GofireConfig) (JobManager, error) {
	var sqlDB *sql.DB
	var redisClient *redis.Client

	switch cfg.StorageDriver {
	case config.Postgres:
		sqlDB = setupPostgres(cfg.PostgresConfig.ConnectionUrl)
	case config.Redis:
		redisClient = setupRedis(cfg.RedisConfig.Address, cfg.RedisConfig.Password, cfg.RedisConfig.DB)
		defer redisClient.Close()
	default:
		return nil, fmt.Errorf("unsupported driver: %v", cfg.StorageDriver)
	}

	jobHandler := NewJobHandler()
	for _, handler := range cfg.Handlers {
		if err := jobHandler.Register(handler.MethodName, handler.Func); err != nil {
			return nil, err
		}
	}

	managers, err := createJobManagers(cfg, sqlDB, redisClient, jobHandler)
	if err != nil {
		return nil, err
	}

	if err := db.Init(cfg.PostgresConfig.ConnectionUrl, managers.LockMgr); err != nil {
		return nil, err
	}

	createDashboardUser(ctx, &cfg, managers.UserRepo)

	if cfg.DashboardAuthEnabled {
		go func() {
			router := web.NewRouteHandler(managers.EnqueuedJobRepo, managers.UserRepo, managers.CronJobRepo, cfg.SecretKey, cfg.DashboardAuthEnabled, cfg.DashboardPort)
			router.Serve()
		}()
	}
	jm := NewJobManager(managers.EnqueuedJobRepo, managers.CronJobRepo, jobHandler)

	jobCtx, cancel := context.WithCancel(ctx)
	jm.cancel = cancel

	jm.wg.Add(2)

	go func() {
		defer jm.wg.Done()
		go managers.EnqueueScheduler.Start(jobCtx, cfg.EnqueueInterval, cfg.WorkerCount, cfg.BatchSize)
	}()

	go func() {
		defer jm.wg.Done()
		go managers.CronJobManager.Start(jobCtx, cfg.ScheduleInterval, cfg.WorkerCount, cfg.BatchSize)
	}()

	return jm, nil
}

func createDashboardUser(ctx context.Context, cfg *config.GofireConfig, repo repository.UserRepository) {
	if cfg.DashboardUserName != "" && cfg.DashboardPassword != "" {
		if user, err := repo.FindByUsername(ctx, cfg.DashboardUserName); err == nil && user == nil {
			repo.Create(ctx, cfg.DashboardUserName, cfg.DashboardPassword)
		} else {
			log.Print(err)
		}
	}
}
