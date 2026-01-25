package types

import (
	"github.com/RezaEskandarii/gofire/internal/state"
	"time"
)

type JobResult struct {
	JobID       int64
	Err         error
	Attempts    int
	MaxAttempts int
	Status      state.JobStatus
	RanAt       time.Time
	NextRun     time.Time
}
