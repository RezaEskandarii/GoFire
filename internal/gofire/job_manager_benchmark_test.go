package gofire

import (
	"context"
	"fmt"
	"gofire/internal/models/config"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

var jobManager JobManager
var cfg *config.GofireConfig

func init() {

	postgresURL := strings.TrimSpace(os.Getenv("gofire_pg_url"))
	if postgresURL == "" {
		panic("could not find job_manager_pg_url is os env list")
	}

	cfg = config.NewGofireConfig("montreal").
		WithEnqueueInterval(2).
		WithScheduleIntervalInterval(2).
		WithWorkerCount(15).
		WithBatchSize(500).
		WithPostgresConfig(config.PostgresConfig{ConnectionUrl: postgresURL})
	cfg = cfg.WithRabbitMQConfig(config.RabbitMQConfig{
		URL:         "amqp://guest:guest@localhost:5672/",
		Exchange:    "gofire_exchange",
		Queue:       "gofire_jobs",
		RoutingKey:  "jobs.enqueue",
		ContentType: "application/json",
	}).WithWriteToRabbitMQueue(true)

	for i := 0; i < 20; i++ {
		jobName := fmt.Sprintf("job-%d", i)
		cfg.RegisterHandler(config.MethodHandler{
			MethodName: jobName,
			Func: func(args ...any) error {
				to := args[0].(string)
				message := args[1].(string)
				return sendSms(to, message, false)
			},
		})
	}

	var err error
	jobManager, err = SetUp(context.Background(), *cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to set up job manager: %v", err))
	}
}

func BenchmarkEnqueue(b *testing.B) {

	b.StartTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobName := "job-5"
	enqueueAt := time.Now().Add(3 * time.Second)
	args := []any{"1234567890", "This is a test message"}

	for i := 0; i < b.N; i++ {
		_, err := jobManager.Enqueue(ctx, jobName, enqueueAt, args...)
		if err != nil {
			b.Fatalf("enqueue failed at iteration %d: %v", i, err)
		}
	}
}
func BenchmarkEnqueueParallel(b *testing.B) {

	for i := 0; i < 20; i++ {
		jobName := fmt.Sprintf("job-%d", i)
		cfg.RegisterHandler(config.MethodHandler{
			MethodName: jobName,
			Func: func(args ...any) error {
				to := args[0].(string)
				message := args[1].(string)
				return sendSms(to, message, false)
			},
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobName := "job-5"
	enqueueAt := time.Now().Add(3 * time.Second)
	args := []any{"1234567890", "This is a test message"}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := jobManager.Enqueue(ctx, jobName, enqueueAt, args...)
			if err != nil {
				b.Errorf("enqueue failed: %v", err)
			}
		}
	})
}

func sendSms(to string, message string, writeToLog bool) error {
	if writeToLog {
		log.Printf("Sending SMS to %s: %s\n", to, message)
	}
	return nil
}
