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

// New initializes the entire Gofire job scheduling and execution system using the provided GofireConfig.
//
// It creates a single Container (dependency injection container) that wires all services,
// then performs bootstrap (DB migrations, admin user), starts background workers (enqueue processor,
// cron processor, optional queue sync), and optionally the web dashboard.
//
// This is the single entry point for application initialization.
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

	// Single container - creates all dependencies once
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

	// Start background workers (enqueue processor, cron processor)
	jobCtx, cancel := context.WithCancel(ctx)
	container.JobManager.Cancel = cancel
	container.JobManager.Wg.Add(2)

	go func() {
		defer container.JobManager.Wg.Done()
		if err := container.EnqueueScheduler.Start(jobCtx, cfg.EnqueueInterval, cfg.WorkerCount, cfg.BatchSize); err != nil {
			log.Printf("EnqueueScheduler failed: %v", err)
		}
	}()

	go func() {
		defer container.JobManager.Wg.Done()
		if err := container.CronJobManager.Start(jobCtx, cfg.ScheduleInterval, cfg.WorkerCount, cfg.BatchSize); err != nil {
			log.Printf("CronJobManager failed: %v", err)
		}
	}()

	// Start queue-storage sync worker if RabbitMQ is enabled
	if cfg.UseQueueWriter {
		go func() {
			if err := container.EnqueueScheduler.StartQueueAndStorageSyncWorker(jobCtx, cfg.RabbitMQConfig.Queue, true); err != nil {
				log.Printf("Queue sync worker failed: %v", err)
			}
		}()
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
