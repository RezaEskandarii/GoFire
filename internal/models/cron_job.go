package models

import (
	"encoding/json"
	"github.com/RezaEskandarii/gofire/internal/state"
	"time"
)

import "database/sql"

type CronJob struct {
	ID         int64
	Name       string
	Payload    json.RawMessage
	Status     state.JobStatus
	LastError  sql.NullString
	LockedBy   *string
	LockedAt   *time.Time
	CreatedAt  time.Time
	LastRunAt  *time.Time
	NextRunAt  time.Time
	IsActive   bool
	Expression string
}
