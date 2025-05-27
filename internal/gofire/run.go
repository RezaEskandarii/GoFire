package gofire

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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

	jobHandler := NewJobHandler()
	for _, handler := range cfg.Handlers {
		if err := jobHandler.Register(handler.MethodName, handler.Func); err != nil {
			return err
		}
	}

	managers, err := createJobManagers(cfg, sqlDB, redisClient, jobHandler)
	if err != nil {
		return err
	}

	if err := db.Init(cfg.PostgresConfig.ConnectionUrl, managers.LockMgr); err != nil {
		return err
	}

	go managers.EnqueueScheduler.Start(ctx, cfg.Interval, cfg.WorkerCount, cfg.BatchSize)
	go managers.CronJobManager.Start(ctx, 60, cfg.WorkerCount, cfg.BatchSize)

	if cfg.EnableDashboard {
		router := web.NewRouteHandler(managers.EnqueuedJobRepo)
		router.Serve(cfg.DashboardPort)
	}

	return nil
}
