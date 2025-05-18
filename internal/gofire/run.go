package gofire

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gofire/internal/app"
	"gofire/web"
)

type MethodHandler struct {
	MethodName string
	Func       func([]interface{}) error
}

type Config struct {
	Connection           string
	DashboardPort        int
	DashboardAuthEnabled bool
	DashboardUserName    string
	DashboardPassword    string
	Instance             string
	EnableDashboard      bool
	Handlers             []MethodHandler
	Driver               StorageDriver
}

func (c Config) RegisterHandler(handler MethodHandler) Config {
	c.Handlers = append(c.Handlers, handler)
	return c
}

func Run(ctx context.Context, config Config) error {
	var sqlDB *sql.DB
	var redisClient *redis.Client

	switch config.Driver {
	case Postgres:
		sqlDB = setupPostgres(config.Connection)
		defer sqlDB.Close()
	case Redis:
		redisClient = setupRedis(ctx, config.Connection)
		defer redisClient.Close()
	default:
		return fmt.Errorf("unsupported driver: %v", config.Driver)
	}

	repo := NewEnqueuedJobRepository(config.Driver, sqlDB, redisClient)
	lockMgr := NewDistributedLockManager(config.Driver, sqlDB, redisClient)

	scheduler := app.NewScheduler(repo, lockMgr, config.Instance)

	for _, handler := range config.Handlers {
		scheduler.RegisterHandler(handler.MethodName, handler.Func)
	}

	go scheduler.ProcessEnqueues(ctx)

	if config.EnableDashboard {
		router := web.NewRouteHandler(repo)
		router.Serve(config.DashboardPort)
	}

	return nil
}
