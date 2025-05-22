package models

import "gofire/internal/state"

type JobResult struct {
	JobID       int64
	Err         error
	Attempts    int
	MaxAttempts int
	Status      state.JobStatus
}
