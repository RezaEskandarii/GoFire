package config

import (
	"testing"
)

func TestStorageDriver_String(t *testing.T) {
	tests := []struct {
		name     string
		driver   StorageDriver
		expected string
	}{
		{
			name:     "Postgres driver",
			driver:   Postgres,
			expected: "postgres",
		},
		{
			name:     "Redis driver",
			driver:   Redis,
			expected: "redis",
		},
		{
			name:     "Unknown driver",
			driver:   StorageDriver(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.driver.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewGofireConfig(t *testing.T) {
	instance := "test-instance"
	config := NewGofireConfig(instance)

	if config.Instance != instance {
		t.Errorf("NewGofireConfig() Instance = %v, want %v", config.Instance, instance)
	}
	if config.EnqueueInterval != DefaultEnqueueInterval {
		t.Errorf("NewGofireConfig() EnqueueInterval = %v, want %v", config.EnqueueInterval, DefaultEnqueueInterval)
	}
	if config.WorkerCount != DefaultWorkerCount {
		t.Errorf("NewGofireConfig() WorkerCount = %v, want %v", config.WorkerCount, DefaultWorkerCount)
	}
	if config.StorageDriver != DefaultStorageDriver {
		t.Errorf("NewGofireConfig() StorageDriver = %v, want %v", config.StorageDriver, DefaultStorageDriver)
	}
	if config.BatchSize != DefaultBatchSize {
		t.Errorf("NewGofireConfig() BatchSize = %v, want %v", config.BatchSize, DefaultBatchSize)
	}
}

func TestGofireConfig_WithDashboardPort(t *testing.T) {
	config := NewGofireConfig("test")
	port := 8080

	result := config.WithDashboardPort(port)

	if result.DashboardPort != port {
		t.Errorf("WithDashboardPort() DashboardPort = %v, want %v", result.DashboardPort, port)
	}
	if !result.EnableDashboard {
		t.Error("WithDashboardPort() EnableDashboard = false, want true")
	}
}

func TestGofireConfig_WithDashboardAuth(t *testing.T) {
	config := NewGofireConfig("test")
	username := "testuser"
	password := "testpass"

	result := config.WithAdminDashboardConfig(username, password)

	if !result.DashboardAuthEnabled {
		t.Error("WithAdminDashboardConfig() DashboardAuthEnabled = false, want true")
	}
	if result.DashboardUserName != username {
		t.Errorf("WithAdminDashboardConfig() DashboardUserName = %v, want %v", result.DashboardUserName, username)
	}
	if result.DashboardPassword != password {
		t.Errorf("WithAdminDashboardConfig() DashboardPassword = %v, want %v", result.DashboardPassword, password)
	}
}

func TestGofireConfig_WithPostgresConfig(t *testing.T) {
	config := NewGofireConfig("test")
	pgConfig := PostgresConfig{ConnectionUrl: "test-url"}

	// Test with Postgres driver
	config.StorageDriver = Postgres
	result := config.WithPostgresConfig(pgConfig)

	if result.StorageDriver != Postgres {
		t.Errorf("WithPostgresConfig() StorageDriver = %v, want %v", result.StorageDriver, Postgres)
	}
	if result.PostgresConfig.ConnectionUrl != pgConfig.ConnectionUrl {
		t.Errorf("WithPostgresConfig() ConnectionUrl = %v, want %v", result.PostgresConfig.ConnectionUrl, pgConfig.ConnectionUrl)
	}

	// Test panic with non-Postgres driver
	config.StorageDriver = Redis
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithPostgresConfig() did not panic with Redis driver")
		}
	}()
	config.WithPostgresConfig(pgConfig)
}

func TestGofireConfig_WithRedisConfig(t *testing.T) {
	config := NewGofireConfig("test")
	redisConfig := RedisConfig{
		Address:  "localhost:6379",
		Password: "password",
		DB:       0,
	}

	// Test with Redis driver
	config.StorageDriver = Redis
	result := config.WithRedisConfig(redisConfig)

	if result.StorageDriver != Redis {
		t.Errorf("WithRedisConfig() StorageDriver = %v, want %v", result.StorageDriver, Redis)
	}
	if result.RedisConfig.Address != redisConfig.Address {
		t.Errorf("WithRedisConfig() Address = %v, want %v", result.RedisConfig.Address, redisConfig.Address)
	}
	if result.RedisConfig.Password != redisConfig.Password {
		t.Errorf("WithRedisConfig() Password = %v, want %v", result.RedisConfig.Password, redisConfig.Password)
	}
	if result.RedisConfig.DB != redisConfig.DB {
		t.Errorf("WithRedisConfig() DB = %v, want %v", result.RedisConfig.DB, redisConfig.DB)
	}

	// Test panic with non-Redis driver
	config.StorageDriver = Postgres
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithRedisConfig() did not panic with Postgres driver")
		}
	}()
	config.WithRedisConfig(redisConfig)
}

func TestGofireConfig_WithWorkerCount(t *testing.T) {
	config := NewGofireConfig("test")
	count := 10

	result := config.WithWorkerCount(count)

	if result.WorkerCount != count {
		t.Errorf("WithWorkerCount() WorkerCount = %v, want %v", result.WorkerCount, count)
	}
}

func TestGofireConfig_WithInterval(t *testing.T) {
	config := NewGofireConfig("test")
	interval := 30

	result := config.WithEnqueueInterval(interval)

	if result.EnqueueInterval != interval {
		t.Errorf("WithEnqueueInterval() EnqueueInterval = %v, want %v", result.EnqueueInterval, interval)
	}
}

func TestGofireConfig_WithBatchSize(t *testing.T) {
	config := NewGofireConfig("test")
	batchSize := 200

	result := config.WithBatchSize(batchSize)

	if result.BatchSize != batchSize {
		t.Errorf("WithBatchSize() BatchSize = %v, want %v", result.BatchSize, batchSize)
	}
}

func TestGofireConfig_RegisterHandler(t *testing.T) {
	config := NewGofireConfig("test")
	handler := MethodHandler{
		MethodName: "test_method",
		Func: func(args []interface{}) error {
			return nil
		},
	}

	result := config.RegisterHandler(handler)

	if len(result.Handlers) != 1 {
		t.Errorf("RegisterHandler() len(Handlers) = %v, want %v", len(result.Handlers), 1)
	}
	if result.Handlers[0].MethodName != handler.MethodName {
		t.Errorf("RegisterHandler() MethodName = %v, want %v", result.Handlers[0].MethodName, handler.MethodName)
	}
}
