package gofire

import (
	"context"
	"github.com/RezaEskandarii/gofire/di"
	"github.com/RezaEskandarii/gofire/types/config"
	"log"
)

var serverInitialized = false

// AddServer launches goroutines for processing enqueued and cron jobs.
// Uses the JobManager wait group to track job reader lifecycle.
func AddServer(ctx context.Context, cfg *config.GofireConfig) error {

	if serverInitialized {
		panic("server is already initialized")
	}

	serverInitialized = true

	jobHandler, dependencies, jobManager, err := di.GetDependencies(cfg)
	if err != nil {
		return err
	}

	jobCtx, cancel := context.WithCancel(ctx)

	jobManager.Cancel = cancel
	jobManager.Wg.Add(2) // Add the number of job reader goroutines to wait for

	for _, handler := range cfg.Handlers {
		if err := jobHandler.Register(handler.JobName, handler.Func); err != nil {
			return err
		}
	}

	// ---------------------------------------------------------------------------------------------
	// Start Enqueue Scheduler in a separate goroutine
	// Responsible for processing enqueued jobs periodically.
	// ---------------------------------------------------------------------------------------------
	go func() {
		defer jobManager.Wg.Done()

		go func() {
			if err := dependencies.EnqueueScheduler.Start(jobCtx, cfg.EnqueueInterval, cfg.WorkerCount, cfg.BatchSize); err != nil {
				log.Printf("EnqueueScheduler failed to start: %v", err)
			}
		}()
	}()

	// ---------------------------------------------------------------------------------------------
	// Start Cron Job Manager in a separate goroutine
	// Responsible for handling cron-based scheduled jobs.
	// ---------------------------------------------------------------------------------------------
	go func() {
		defer jobManager.Wg.Done()

		go func() {
			if err := dependencies.CronJobManager.Start(jobCtx, cfg.ScheduleInterval, cfg.WorkerCount, cfg.BatchSize); err != nil {
				log.Printf("CronJobManager failed to start: %v", err)
			}
		}()
	}()

	// ---------------------------------------------------------------------------------------------
	// Start Queue and Storage Sync Worker if queue writer is enabled
	// Syncs storage with message broker (RabbitMQ).
	// ---------------------------------------------------------------------------------------------
	if cfg.UseQueueWriter {
		go func() {
			if err := dependencies.EnqueueScheduler.StartQueueAndStorageSyncWorker(jobCtx, cfg.RabbitMQConfig.Queue, cfg.UseQueueWriter); err != nil {
				log.Printf("EnqueueScheduler failed to startQueueAndStorageSyncWorker: %v", err)
			}
		}()
	}

	return nil
}
