package types

import (
	"database/sql"
	"encoding/json"
	"github.com/RezaEskandarii/gofire/internal/state"
	"time"
)

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
