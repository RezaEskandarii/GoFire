package config

import "fmt"

type StorageDriver int

const (
	Postgres StorageDriver = iota + 1
	Redis                  // seconds
)

const (
	DefaultWorkerCount   = 5
	DefaultInterval      = 15
	DefaultStorageDriver = Postgres
	DefaultBatchSize     = 100
)

func (d StorageDriver) String() string {
	switch d {
	case Redis:
		return "redis"
	case Postgres:
		return "postgres"
	}
	return "unknown"
}

type MethodHandler struct {
	MethodName string
	Func       func(args []interface{}) error
}

type PostgresConfig struct {
	ConnectionUrl string
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type GofireConfig struct {
	DashboardPort        int
	DashboardAuthEnabled bool
	DashboardUserName    string
	DashboardPassword    string
	Instance             string
	EnableDashboard      bool
	Handlers             []MethodHandler
	StorageDriver        StorageDriver
	WorkerCount          int
	Interval             int
	BatchSize            int
	// Storage Configs
	PostgresConfig PostgresConfig
	RedisConfig    RedisConfig
}

func NewGofireConfig(instance string) *GofireConfig {
	return &GofireConfig{
		Instance:      instance,
		Interval:      DefaultInterval,
		WorkerCount:   DefaultWorkerCount,
		StorageDriver: DefaultStorageDriver,
		BatchSize:     DefaultBatchSize,
	}
}

func (c *GofireConfig) WithDashboardPort(port int) *GofireConfig {
	c.DashboardPort = port
	c.EnableDashboard = true
	return c
}

func (c *GofireConfig) WithDashboardAuth(username, password string) *GofireConfig {
	c.DashboardAuthEnabled = true
	c.DashboardUserName = username
	c.DashboardPassword = password
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

func (c *GofireConfig) WithInterval(seconds int) *GofireConfig {
	c.Interval = seconds
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
