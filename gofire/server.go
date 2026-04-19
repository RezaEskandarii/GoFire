package gofire

import (
	"context"
	"fmt"
	"github.com/RezaEskandarii/gofire/app"
	"github.com/RezaEskandarii/gofire/client"
	"github.com/RezaEskandarii/gofire/types/config"
	"log"
)

// AddServer starts background workers that execute enqueued and cron jobs.
func AddServer(ctx context.Context, cfg *config.GofireConfig) (*client.JobManager, error) {
	container, err := app.NewContainer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return AddServerWithContainer(ctx, container)
}

// AddServerWithContainer starts workers using an already-built dependency container.
// Useful when you want a single process to both bootstrap/dashboard and run workers
// without creating duplicate DB connections.
func AddServerWithContainer(ctx context.Context, container *app.Container) (*client.JobManager, error) {
	if container == nil || container.Config == nil || container.JobManager == nil {
		return nil, fmt.Errorf("nil container")
	}
	cfg := container.Config

	// Register all job handlers from config
	for _, handler := range cfg.Handlers {
		if err := container.JobHandler.Register(handler.JobName, handler.Func); err != nil {
			return nil, err
		}
	}

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

	if cfg.UseQueueWriter {
		go func() {
			if err := container.EnqueueScheduler.StartQueueAndStorageSyncWorker(jobCtx, cfg.RabbitMQConfig.Queue, true); err != nil {
				log.Printf("Queue sync worker failed: %v", err)
			}
		}()
	}

	return container.JobManager, nil
}
