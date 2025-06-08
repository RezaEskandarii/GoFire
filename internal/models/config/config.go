package config

import "fmt"

type GofireConfig struct {
	DashboardPort int // Port number used to serve the monitoring dashboard (e.g., 8080)

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

	// WriteToRabbitQueue determines whether jobs should be first sent to RabbitMQ queue.
	// If true, jobs are enqueued in RabbitMQ before being processed and batch-inserted into the database.
	WriteToRabbitQueue bool

	MQDriver MQDriver

	// RabbitMQConfig holds the configuration settings for connecting to RabbitMQ,
	// such as connection URL, queue names, and other relevant parameters.
	RabbitMQConfig *RabbitMQConfig
}

// MethodHandler holds the name and actual function of a job handler.
type MethodHandler struct {
	MethodName string                  // Name used to identify the handler (e.g., "SendEmail")
	Func       func(args ...any) error // The function to execute for this handler
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	ConnectionUrl string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Address  string // Redis server address (e.g., "localhost:6379")
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

// NewGofireConfig creates a new instance of GofireConfig with default values.
// Only the 'Instance' name is required; other fields use predefined defaults.
func NewGofireConfig(instance string) *GofireConfig {
	return &GofireConfig{
		Instance:         instance,
		EnqueueInterval:  DefaultEnqueueInterval,
		WorkerCount:      DefaultWorkerCount,
		StorageDriver:    DefaultStorageDriver,
		BatchSize:        DefaultBatchSize,
		ScheduleInterval: DefaultCronJobInterval,
		RabbitMQConfig:   &RabbitMQConfig{},
	}
}

func (c *GofireConfig) WithAdminDashboardConfig(username, password, secretKey string, port int) *GofireConfig {
	c.DashboardAuthEnabled = true
	c.DashboardUserName = username
	c.DashboardPassword = password
	c.SecretKey = secretKey
	c.DashboardPort = port
	return c
}

func (c *GofireConfig) WithPostgresConfig(pg PostgresConfig) *GofireConfig {
	if c.StorageDriver != Postgres {
		panic(fmt.Sprintf("Cannot set Postgres config when driver is %s", c.StorageDriver.String()))
	}
	c.StorageDriver = Postgres
	c.PostgresConfig = pg
	return c
}

func (c *GofireConfig) WithRedisConfig(r RedisConfig) *GofireConfig {
	if c.StorageDriver != Redis {
		panic(fmt.Sprintf("Cannot set Redis config when driver is %s", c.StorageDriver.String()))
	}
	c.StorageDriver = Redis
	c.RedisConfig = r
	return c
}

func (c *GofireConfig) WithWorkerCount(n int) *GofireConfig {
	c.WorkerCount = n
	return c
}

func (c *GofireConfig) WithEnqueueInterval(seconds int) *GofireConfig {
	c.EnqueueInterval = seconds
	return c
}

func (c *GofireConfig) WithScheduleIntervalInterval(seconds int) *GofireConfig {
	c.ScheduleInterval = seconds
	return c
}

func (c *GofireConfig) WithBatchSize(batchSize int) *GofireConfig {
	c.BatchSize = batchSize
	return c
}

func (c *GofireConfig) RegisterHandler(handler MethodHandler) *GofireConfig {
	c.Handlers = append(c.Handlers, handler)
	return c
}

// WithWriteToRabbitMQueue enables or disables writing jobs first to RabbitMQ queue.
// When enabled (writeToQueue = true), jobs are initially pushed to RabbitMQ,
// and later consumed in batches for bulk writing into the database.
// This approach helps decouple job submission from database writes,
// improving throughput and scalability.
func (c *GofireConfig) WithWriteToRabbitMQueue(writeToQueue bool) *GofireConfig {
	c.WriteToRabbitQueue = writeToQueue
	c.MQDriver = RabbitMQ
	return c
}

func (c *GofireConfig) WithRabbitMQConfig(cfg RabbitMQConfig) *GofireConfig {
	c.RabbitMQConfig = &cfg
	c.WriteToRabbitQueue = true
	c.MQDriver = RabbitMQ
	return c
}
