package gofire

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gofire/internal/app"
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

	repo := NewEnqueuedJobRepository(cfg.StorageDriver, sqlDB, redisClient)
	lockMgr := NewDistributedLockManager(cfg.StorageDriver, sqlDB, redisClient)

	scheduler := app.NewScheduler(repo, lockMgr, cfg.Instance)

	for _, handler := range cfg.Handlers {
		scheduler.RegisterHandler(handler.MethodName, handler.Func)
	}

	//duration := time.Now().Add(time.Minute)
	//for i := 0; i < 10000; i++ {
	//	phone := fmt.Sprintf("093740528_%d", i)
	//	message := fmt.Sprintf("message_%d", i)
	//	scheduler.Enqueue(ctx, "send_sms", duration, []interface{}{phone, message})
	//	fmt.Println(phone)
	//}

	go scheduler.ProcessEnqueues(ctx, cfg.Interval, cfg.WorkerCount)

	if cfg.EnableDashboard {
		router := web.NewRouteHandler(repo)
		router.Serve(cfg.DashboardPort)
	}

	return nil
}
