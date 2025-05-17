package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Lock struct {
	db *sql.DB
}

func NewLock(db *sql.DB) Lock {
	return Lock{
		db: db,
	}
}

func (l *Lock) AcquirePostgresDistributedLock(lockID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := l.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil
}

func (l *Lock) ReleasePostgresDistributedLock(lockID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := l.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}
