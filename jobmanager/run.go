package jobmanager

import (
	"context"
	"github.com/RezaEskandarii/gofire/app"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/internal/db"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/types/config"
	"github.com/RezaEskandarii/gofire/web"
	"log"
	"runtime"
)

// New creates a JobManager for enqueue/schedule operations and (optionally) starts the dashboard.
//
// IMPORTANT: This does NOT start background workers. To process jobs (like Hangfire Server),
// run gofire.AddServer in the dedicated worker pod(s).
//
// Parameters:
//   - ctx: context used for cancellation and timeout propagation to background workers.
//   - cfg: full configuration of the system (storage, handlers, worker count, dashboard options).
//
// Returns:
//   - JobManager: a configured job manager ready to enqueue, schedule, and monitor jobs.
//   - error: any failure that prevents full system setup.
func New(ctx context.Context, cfg *config.GofireConfig) (*client.JobManager, error) {
	log.Printf("GOMAXPROCS Is: %d\n", runtime.GOMAXPROCS(0))

	defer func() {
		if r := recover(); r != nil {
			log.Println("panic during init:", r)
		}
	}()

	// Creates dependencies (DB, stores, handlers, etc.). No workers are started here.
	container, err := app.NewContainer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Register all job handlers from config
	for _, handler := range cfg.Handlers {
		if err := container.JobHandler.Register(handler.JobName, handler.Func); err != nil {
			return nil, err
		}
	}

	// Bootstrap: DB migrations (protected by distributed lock)
	if err := db.Init(cfg.PostgresConfig.ConnectionUrl, container.LockManager); err != nil {
		return nil, err
	}

	// Bootstrap: Create dashboard admin if configured
	if err := createDashboardAdminIfConfigured(ctx, cfg, container.UserStore); err != nil {
		return nil, err
	}

	// Start web dashboard if auth is enabled
	if cfg.DashboardAuthEnabled {
		go func() {
			router := web.NewRouteHandler(
				container.EnqueuedJobStore,
				container.UserStore,
				container.CronJobStore,
				cfg.SecretKey,
				cfg.DashboardAuthEnabled,
				cfg.DashboardPort,
			)
			if err := router.Serve(); err != nil {
				log.Printf("web dashboard failed: %v", err)
			}
		}()
	}

	return container.JobManager, nil
}

func createDashboardAdminIfConfigured(ctx context.Context, cfg *config.GofireConfig, userStore store.UserStore) error {
	if cfg.DashboardUserName == "" || cfg.DashboardPassword == "" {
		return nil
	}
	user, err := userStore.FindByUsername(ctx, cfg.DashboardUserName)
	if err != nil {
		return err
	}
	if user != nil {
		return nil // Admin already exists
	}
	_, err = userStore.Create(ctx, cfg.DashboardUserName, cfg.DashboardPassword)
	return err
}
