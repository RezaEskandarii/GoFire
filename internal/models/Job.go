package models

import "time"

type Job struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Args        []any     `json:"args"`
	ScheduledAt time.Time `json:"scheduled_at"`
}
