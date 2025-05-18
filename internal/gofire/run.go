package gofire

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"gofire/internal/app"
	"gofire/internal/db"
	"gofire/internal/lock"
	"gofire/internal/repository"
	"gofire/web"
	"log"
)

type MethodHandler struct {
	MethodName string
	Func       func([]interface{}) error
}

type Config struct {
	PostgresConnection   string
	DashboardPort        int
	DashboardAuthEnabled bool
	DashboardUserName    string
	DashboardPassword    string
	Instance             string
	EnableDashboard      bool
	Handlers             []MethodHandler
	Driver               DatabaseDriver
}

func (c Config) RegisterHandler(handler MethodHandler) Config {
	c.Handlers = append(c.Handlers, handler)
	return c
}

func Run(ctx context.Context, config Config) error {
	sqlDB := setupDB(config.PostgresConnection)
	defer sqlDB.Close()

	if err := db.Init(config.PostgresConnection); err != nil {
		log.Println(err)
		return err
	}

	enqueuedJobRepository := repository.NewPostgresEnqueuedJobRepository(sqlDB)
	distributedLock := lock.NewPostgresDistributedLockManager(sqlDB)

	enqueueScheduler := app.NewScheduler(&enqueuedJobRepository, &distributedLock, config.Instance)

	for _, handler := range config.Handlers {
		enqueueScheduler.RegisterHandler(handler.MethodName, handler.Func)
	}

	go enqueueScheduler.ProcessEnqueues(ctx)

	if config.EnableDashboard {
		router := web.NewRouteHandler(&enqueuedJobRepository)
		router.Serve(config.DashboardPort)
	}

	return nil
}

func setupDB(connStr string) *sql.DB {
	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	return sqlDB
}
