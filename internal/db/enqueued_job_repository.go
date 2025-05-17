package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"gofire/internal/constants"
	"gofire/internal/task"
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

func (r *JobRepository) Insert(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error) {

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

func (r *JobRepository) FetchDueJobs(ctx context.Context, limit int) ([]task.EnqueuedJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, payload, attempts, max_attempts
		FROM gofire_schema.enqueued_jobs
		WHERE scheduled_at <= NOW()
		  AND status = 'queued'
		ORDER BY scheduled_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []task.EnqueuedJob
	for rows.Next() {
		var job task.EnqueuedJob
		if err := rows.Scan(&job.ID, &job.Name, &job.Payload, &job.Attempts, &job.MaxAttempts); err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *JobRepository) LockJob(ctx context.Context, jobID int, lockedBy string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET locked_at = NOW(), locked_by = $1, status = 'processing'
		WHERE id = $2 AND status = 'queued'
	`, lockedBy, jobID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (r *JobRepository) MarkSuccess(ctx context.Context, jobID int) {
	r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET status = 'succeeded', executed_at = NOW(), finished_at = NOW()
		WHERE id = $1
	`, jobID)
}

func (r *JobRepository) MarkFailure(ctx context.Context, jobID int, errMsg string, attempts int, maxAttempts int) {
	status := task.StatusFailed
	if attempts+1 >= constants.MaxRetryAttempt {
		status = task.StatusDead
	}
	r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET attempts = attempts + 1,
		    last_error = $2,
		    status = $3
		WHERE id = $1
	`, jobID, errMsg, status)
}
