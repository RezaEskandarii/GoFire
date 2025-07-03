package test

import (
	"context"
	"fmt"
	"gofire/internal/gofire"
	"gofire/internal/gofire/test/mocks"
	"gofire/internal/models/config"
	"log"
	"os"
	"strings"
)

var testJobManager *gofire.JobManager
var benchJobManager *gofire.JobManager
var cfg *config.GofireConfig

func init() {

	postgresURL := strings.TrimSpace(os.Getenv("gofire_pg_url"))
	if postgresURL == "" {
		panic("could not find job_manager_pg_url os env")
	}
	var err error

	cfg, err = config.NewGofireConfig("accounting-api",
		config.WithEnqueueInterval(6),
		config.WithScheduleInterval(60),
		config.WithWorkerCount(15),
		config.WithBatchSize(500),
		config.WithPostgresConfig(config.PostgresConfig{ConnectionUrl: postgresURL}),
	)

	//cfg = cfg.WithRabbitMQConfig(config.RabbitMQConfig{
	//	URL:         "amqp://guest:guest@localhost:5672/",
	//	Exchange:    "gofire_exchange",
	//	Queue:       "gofire_jobs",
	//	RoutingKey:  "jobs.enqueue",
	//	ContentType: "application/json",
	//}).UseRabbitMQueueWriter(true)

	for i := 0; i < 20; i++ {
		jobName := fmt.Sprintf("job-%d", i)
		cfg.RegisterHandler(config.MethodHandler{
			JobName: jobName,
			Func: func(args ...any) error {
				to := args[0].(string)
				message := args[1].(string)
				return sendSms(to, message, false)
			},
		})
	}

	benchJobManager, err = gofire.BootJobManager(context.Background(), *cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to set up job manager: %v", err))
	}

	mockEnqueuedStore := mocks.NewMockEnqueuedJobStore()
	mockCronJobStore := mocks.NewMockCronJobStore()

	mockDistributedLockManager := mocks.NewMockDistributedLockManager()
	mockMessageBroker := mocks.NewMockMessageBroker(1000)

	testJobManager = gofire.NewJobManager(
		mockEnqueuedStore,
		mockCronJobStore,
		createTestJobHandler(),
		mockDistributedLockManager,
		mockMessageBroker,
		cfg.UseQueueWriter,
		cfg.RabbitMQConfig.Queue,
	)

}

func sendSms(to string, message string, writeToLog bool) error {
	if writeToLog {
		log.Printf("Sending SMS to %s: %s\n", to, message)
	}
	return nil
}

func createTestJobHandler() *gofire.JobHandler {
	jh := gofire.NewJobHandler()
	_ = jh.Register("email", func(args ...any) error {
		return nil
	})
	_ = jh.Register("daily", func(args ...any) error {
		return nil
	})
	_ = jh.Register("heartbeat", func(args ...any) error {
		return nil
	})
	_ = jh.Register("test", func(args ...any) error {
		return nil
	})
	return jh
}
