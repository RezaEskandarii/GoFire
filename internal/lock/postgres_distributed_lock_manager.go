package lock

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PostgresDistributedLockManager struct {
	db *sql.DB
}

func NewPostgresDistributedLockManager(db *sql.DB) *PostgresDistributedLockManager {
	return &PostgresDistributedLockManager{
		db: db,
	}
}

func (l *PostgresDistributedLockManager) Acquire(lockID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := l.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil
}

func (l *PostgresDistributedLockManager) Release(lockID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := l.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}
