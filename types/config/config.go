package config

import (
	"errors"
	"fmt"
	errors2 "github.com/RezaEskandarii/gofire/errors"
)

type GofireConfig struct {
	DashboardPort uint // Port number used to serve the monitoring dashboard (e.g., 8080)

	DashboardUserName    string          // Username required for accessing the dashboard (if auth is enabled)
	DashboardPassword    string          // Password required for accessing the dashboard (if auth is enabled)
	SecretKey            string          // Admin dashboard authentication cookie secret key
	Instance             string          // Unique identifier for this instance (used for distinguishing multiple instances)
	DashboardAuthEnabled bool            // Flag to completely enable or disable the dashboard feature
	Handlers             []MethodHandler // List of registered job/function handlers
	StorageDriver        StorageDriver   // Specifies the storage backend (e.g., Redis, PostgreSQL)
	WorkerCount          int             // Number of concurrent worker goroutines processing jobs
	EnqueueInterval      int             // Interval (in seconds or milliseconds) for enqueueing jobs from storage
	ScheduleInterval     int             // Interval (in seconds) to evaluate cron job schedules

	BatchSize int // Number of jobs fetched from storage per batch

	// Configuration for PostgreSQL storage driver
	PostgresConfig PostgresConfig
	// Configuration for Redis storage driver
	RedisConfig RedisConfig

	// UseQueueWriter determines whether jobs should be first sent to RabbitMQ queue.
	// If true, jobs are enqueued in RabbitMQ before being processed and batch-inserted into the database.
	UseQueueWriter bool

	MQDriver MessageQueueDriver

	// RabbitMQConfig holds the configuration settings for connecting to RabbitMQ,
	// such as connection URL, queue names, and other relevant parameters.
	RabbitMQConfig *RabbitMQConfig
}

// MethodHandler holds the name and actual function of a job handler.
type MethodHandler struct {
	JobName string                  // Name used to identify the handler (e.g., "SendEmail")
	Func    func(args ...any) error // The function to execute for this handler
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	ConnectionUrl string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Address  string // Redis client address (e.g., "localhost:6379")
	Password string // Password for Redis authentication (optional)
	DB       int    // Redis database number to use (e.g., 0 by default)
}

type RabbitMQConfig struct {
	URL         string // For example:  amqp://guest:guest@localhost:5672/
	Exchange    string
	Queue       string
	RoutingKey  string
	ContentType string
}

// ContainerOption type for functional options pattern
type ContainerOption func(*GofireConfig) error

// NewGofireConfig creates a new instance of GofireConfig with default values.
// Only the 'Instance' name is required; other fields use predefined defaults.
func NewGofireConfig(instance string, opts ...ContainerOption) (*GofireConfig, error) {
	cfg := &GofireConfig{
		Instance:         instance,
		EnqueueInterval:  DefaultEnqueueInterval,
		WorkerCount:      DefaultWorkerCount,
		StorageDriver:    DefaultStorageDriver,
		BatchSize:        DefaultBatchSize,
		ScheduleInterval: DefaultCronJobInterval,
		RabbitMQConfig:   &RabbitMQConfig{},
	}
	validationErrs := &errors2.ValidationError{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			validationErrs.Add(err)
		}
	}

	if validationErrs.HasError() {
		return nil, validationErrs
	}
	return cfg, nil
}

func WithAdminDashboardConfig(username, password, secretKey string, port uint) ContainerOption {
	return func(c *GofireConfig) error {
		if username == "" || password == "" || secretKey == "" || port == 0 {
			return errors.New("admin dashboard client: username, password, secretKey, and port are required")
		}

		c.DashboardAuthEnabled = true
		c.DashboardUserName = username
		c.DashboardPassword = password
		c.SecretKey = secretKey
		c.DashboardPort = port
		return nil
	}
}

func WithPostgresConfig(pg PostgresConfig) ContainerOption {
	return func(c *GofireConfig) error {
		if c.StorageDriver != Postgres {
			return fmt.Errorf("cannot set Postgres client when driver is %s", c.StorageDriver.String())
		}
		if pg.ConnectionUrl == "" {
			return errors.New("postgres client: connection URL is required")
		}
		c.StorageDriver = Postgres
		c.PostgresConfig = pg
		return nil
	}
}

func WithWorkerCount(n int) ContainerOption {
	return func(c *GofireConfig) error {
		if n < 1 {
			return errors.New("worker count must be positive")
		}
		c.WorkerCount = n
		return nil
	}
}

func WithEnqueueInterval(seconds int) ContainerOption {
	return func(c *GofireConfig) error {
		if seconds < 1 {
			return errors.New("enqueue interval must be positive")
		}
		c.EnqueueInterval = seconds
		return nil
	}
}

func WithScheduleInterval(seconds int) ContainerOption {
	return func(c *GofireConfig) error {
		if seconds < 1 {
			return errors.New("schedule interval must be positive")
		}
		c.ScheduleInterval = seconds
		return nil
	}
}

func WithBatchSize(batchSize int) ContainerOption {
	return func(c *GofireConfig) error {
		if batchSize < 1 {
			return errors.New("batch size must be positive")
		}
		c.BatchSize = batchSize
		return nil
	}
}

func (c *GofireConfig) RegisterHandler(handler MethodHandler) error {
	if handler.JobName == "" || handler.Func == nil {
		return errors.New("handler must have a job name and function")
	}
	c.Handlers = append(c.Handlers, handler)

	return nil
}

func (c *GofireConfig) RegisterHandlers(handlers []MethodHandler) error {
	for _, h := range handlers {
		if err := c.RegisterHandler(h); err != nil {
			return err
		}
	}
	return nil
}

// UseRabbitMQueueWriter enables or disables writing jobs first to RabbitMQ queue.
// When enabled (writeToQueue = true), jobs are initially pushed to RabbitMQ,
// and later consumed in batches for bulk writing into the database.
// This approach helps decouple job submission from database writes,
// improving throughput and scalability.
func UseRabbitMQueueWriter(writeToQueue bool) ContainerOption {
	return func(c *GofireConfig) error {
		c.UseQueueWriter = writeToQueue
		if writeToQueue {
			c.MQDriver = RabbitMQ
		}
		return nil
	}
}

func WithRabbitMQConfig(cfg RabbitMQConfig) ContainerOption {
	return func(c *GofireConfig) error {
		if cfg.URL == "" {
			return errors.New("rabbitmq client: URL is required")
		}
		c.RabbitMQConfig = &cfg
		c.UseQueueWriter = true
		c.MQDriver = RabbitMQ
		return nil
	}
}
