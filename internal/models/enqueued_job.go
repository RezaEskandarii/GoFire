package models

import (
	"encoding/json"
	"gofire/internal/state"
	"time"
)

type EnqueuedJob struct {
	ID          int64
	Name        string
	Payload     json.RawMessage
	Status      state.JobStatus
	Attempts    int
	MaxAttempts int
	ScheduledAt time.Time
	ExecutedAt  *time.Time
	FinishedAt  *time.Time
	LastError   *string
	LockedBy    *string
	LockedAt    *time.Time
	CreatedAt   time.Time
}
