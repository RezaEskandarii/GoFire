package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) JobRepository {
	return JobRepository{
		db: db,
	}
}

func (r *JobRepository) InsertEnqueuedJob(ctx context.Context, jobName string, scheduledAt time.Time, args ...any) (int64, error) {

	payloadJSON, err := json.Marshal(args)
	if err != nil {
		return -1, err
	}

	query := `
        INSERT INTO gofire_schema.enqueued_jobs (
            name, payload, scheduled_at, max_attempts, created_at
        )
        VALUES ($1, $2, $3, $4, now())
        returning id
    `

	var jobID int64
	err = r.db.QueryRowContext(ctx, query,
		jobName,
		payloadJSON,
		scheduledAt,
		0,
	).Scan(&jobID)

	return jobID, err

}
