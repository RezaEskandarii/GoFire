package jobmanager

import (
	"context"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/di"
	"github.com/RezaEskandarii/gofire/internal/db"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/RezaEskandarii/gofire/types/config"
	"github.com/RezaEskandarii/gofire/web"
	_ "github.com/lib/pq"
	"log"
	"runtime"
)

// New initializes the entire Gofire job scheduling and execution system using the provided GofireConfig.
//
// It dynamically sets up storage backends (PostgreSQL or Redis), registers job handlers,
// initializes the Storesitories and distributed lock manager, and launches all necessary background services,
// including job enqueuing, cron evaluation, and optionally the admin dashboard.
//
// The function performs the following steps:
//  1. Connects to the specified storage backend based on cfg.StorageDriver (Postgres or Redis).
//  2. Registers all job handlers defined in cfg.Handlers.
//  3. Instantiates Storesitories, schedulers, and locking mechanisms using createJobDependency.
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
//   - error: any failure that prevents full system setup (e.g., invalid client, failed connection, migration error).
func New(ctx context.Context, cfg *config.GofireConfig) (*client.JobManager, error) {

	log.Printf("GOMAXPROCS Is: %d\n", runtime.GOMAXPROCS(0))
	// ---------------------------------------------------------------------------------------------
	// Recover from any panic and log the error (for safety in booting process)
	// ---------------------------------------------------------------------------------------------
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	jobHandler, dependencies, jobManager, err := di.GetDependencies(cfg)
	if err != nil {
		return jobManager, err
	}

	// ---------------------------------------------------------------------------------------------
	// Initialize database with lock jobManager (for schema setup or other bootstrapping logic)
	// ---------------------------------------------------------------------------------------------
	if err = db.Init(cfg.PostgresConfig.ConnectionUrl, dependencies.LockMgr); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------------------------
	// If admin user is defined in client, create the dashboard admin account
	// ---------------------------------------------------------------------------------------------
	if err = createDashboardAdminIfConfigured(ctx, cfg, dependencies.UserStore); err != nil {
		return nil, err
	}

	// ---------------------------------------------------------------------------------------------
	// Start dashboard client if authentication is enabled
	// ---------------------------------------------------------------------------------------------
	if cfg.DashboardAuthEnabled {
		runWebServer(dependencies, cfg)
	}

	// ---------------------------------------------------------------------------------------------
	// Create the main JobManager with initialized components
	// ---------------------------------------------------------------------------------------------
	jm := client.NewJobManager(
		dependencies.EnqueuedJobStore,
		dependencies.CronJobStore,
		jobHandler,
		dependencies.LockMgr,
		dependencies.MessageBroker,
		cfg.UseQueueWriter,
		cfg.RabbitMQConfig.Queue,
	)

	// ---------------------------------------------------------------------------------------------
	// Return the fully initialized JobManager
	// ---------------------------------------------------------------------------------------------
	return jm, nil
}

// runWebServer initializes and starts the web client for the dashboard interface in a separate goroutine.
func runWebServer(managers *di.JobDependency, cfg *config.GofireConfig) {
	go func() {
		router := web.NewRouteHandler(managers.EnqueuedJobStore, managers.UserStore, managers.CronJobStore, cfg.SecretKey, cfg.DashboardAuthEnabled, cfg.DashboardPort)
		if err := router.Serve(); err != nil {
			log.Printf("failed to start client: %v", err)
		}
	}()
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
