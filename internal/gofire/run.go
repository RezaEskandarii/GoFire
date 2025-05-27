package gofire

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gofire/internal/app"
	"gofire/internal/db"
	"gofire/internal/models/config"
	"gofire/web"
)

func Run(ctx context.Context, cfg config.GofireConfig) error {
	var sqlDB *sql.DB
	var redisClient *redis.Client

	switch cfg.StorageDriver {
	case config.Postgres:
		sqlDB = setupPostgres(cfg.PostgresConfig.ConnectionUrl)
		defer sqlDB.Close()
	case config.Redis:
		redisClient = setupRedis(cfg.RedisConfig.Address, cfg.RedisConfig.Password, cfg.RedisConfig.DB)
		defer redisClient.Close()
	default:
		return fmt.Errorf("unsupported driver: %v", cfg.StorageDriver)
	}

	repo := CreateEnqueuedJobRepository(cfg.StorageDriver, sqlDB, redisClient)
	lockMgr := CreateDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	db.Init(cfg.PostgresConfig.ConnectionUrl, lockMgr)

	jobHandler := app.NewJobHandler()
	for _, handler := range cfg.Handlers {
		jobHandler.Register(handler.MethodName, handler.Func)
	}

	scheduler := app.NewEnqueueScheduler(repo, lockMgr, jobHandler, cfg.Instance)

	go scheduler.ProcessEnqueues(ctx, cfg.Interval, cfg.WorkerCount, cfg.BatchSize)

	if cfg.EnableDashboard {
		router := web.NewRouteHandler(repo)
		router.Serve(cfg.DashboardPort)
	}

	return nil
}
