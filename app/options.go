package app

import (
	"database/sql"
	"github.com/redis/go-redis/v9"
)

// ContainerOption configures Container creation. Used for testing and customization.
type ContainerOption func(*containerConfig)

type containerConfig struct {
	// Optional: inject custom DB instead of creating from config
	db    *sql.DB
	redis *redis.Client
}

// WithDB injects a custom database connection. Useful for testing.
func WithDB(db *sql.DB) ContainerOption {
	return func(c *containerConfig) {
		c.db = db
	}
}

// WithRedis injects a custom Redis client. Useful for testing.
func WithRedis(redis *redis.Client) ContainerOption {
	return func(c *containerConfig) {
		c.redis = redis
	}
}
