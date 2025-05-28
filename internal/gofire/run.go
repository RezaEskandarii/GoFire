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

func SetUp(ctx context.Context, cfg config.GofireConfig) (JobManager, error) {
	var sqlDB *sql.DB
	var redisClient *redis.Client

	switch cfg.StorageDriver {
	case config.Postgres:
		sqlDB = setupPostgres(cfg.PostgresConfig.ConnectionUrl)
		//defer sqlDB.Close()
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

	go managers.EnqueueScheduler.Start(ctx, cfg.EnqueueInterval, cfg.WorkerCount, cfg.BatchSize)
	go managers.CronJobManager.Start(ctx, 3, cfg.WorkerCount, cfg.BatchSize)

	if cfg.EnableDashboard {
		go func() {
			router := web.NewRouteHandler(managers.EnqueuedJobRepo)
			router.Serve(cfg.DashboardPort)
		}()
	}
	return NewJobManager(managers.EnqueuedJobRepo, managers.CronJobRepo, jobHandler), nil
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
