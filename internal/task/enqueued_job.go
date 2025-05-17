package task

import (
	"encoding/json"
	"time"
)

type EnqueuedJob struct {
	ID          int
	Name        string
	Payload     json.RawMessage
	Status      string
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
