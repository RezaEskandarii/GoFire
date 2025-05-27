package models

import (
	"encoding/json"
	"gofire/internal/state"
	"time"
)

type CronJob struct {
	ID         int64
	Name       string
	Payload    json.RawMessage
	Status     state.JobStatus
	LastError  string
	LockedBy   *string
	LockedAt   *time.Time
	CreatedAt  time.Time
	LastRunAt  time.Time
	NextRunAt  time.Time
	IsActive   bool
	Expression string
}
