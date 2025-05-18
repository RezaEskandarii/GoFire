package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/state"
	"gofire/internal/task"
	"math"
	"time"
)

type PaginationResult[T any] struct {
	Items           []T  `json:"items"`
	TotalItems      int  `json:"total_items"`
	Page            int  `json:"page"`
	PageSize        int  `json:"page_size"`
	TotalPages      int  `json:"total_pages"`
	HasNextPage     bool `json:"has_next_page"`
	HasPreviousPage bool `json:"has_previous_page"`
}

type EnqueuedJobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) EnqueuedJobRepository {
	return EnqueuedJobRepository{
		db: db,
	}
}

func (r *EnqueuedJobRepository) Insert(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error) {

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
func (r *EnqueuedJobRepository) FetchDueJobs(
	ctx context.Context,
	page int,
	pageSize int,
	status *state.JobStatus,
	scheduledBefore *time.Time) (*PaginationResult[task.EnqueuedJob], error) {

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	where := "1=1"
	args := []interface{}{}
	argIndex := 1

	if scheduledBefore != nil {
		where += fmt.Sprintf(" AND scheduled_at <= $%d", argIndex)
		args = append(args, scheduledBefore.Format("2006-01-02"))
		argIndex++
	}

	if status != nil && *status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	countQuery := `SELECT COUNT(*) FROM gofire_schema.enqueued_jobs WHERE ` + where
	selectQuery := `
		SELECT id,
		       name, 
		       payload, 
		       status,
		       attempts,
		       max_attempts,
		       scheduled_at, 
		       executed_at,
		       finished_at,
		       last_error,
		       locked_by,
		       locked_at, 
		       created_at
		FROM gofire_schema.enqueued_jobs
		WHERE ` + where + fmt.Sprintf(" ORDER BY scheduled_at ASC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	var totalItems int
	err := r.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []task.EnqueuedJob
	for rows.Next() {
		job, err := r.mapSqlRowsToJob(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, *job)
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	result := &PaginationResult[task.EnqueuedJob]{
		Items:           jobs,
		TotalItems:      totalItems,
		Page:            page,
		PageSize:        pageSize,
		TotalPages:      totalPages,
		HasNextPage:     page < totalPages,
		HasPreviousPage: page > 1,
	}

	return result, nil
}

func (r *EnqueuedJobRepository) LockJob(ctx context.Context, job *task.EnqueuedJob, lockedBy string) (bool, error) {
	job.Status = state.StatusProcessing
	res, err := r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET locked_at = NOW(), 
		    locked_by = $1,
		    status = 'processing'
		WHERE id = $2 AND status = 'queued'
	`, lockedBy, job.ID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (r *EnqueuedJobRepository) MarkSuccess(ctx context.Context, jobID int) {
	r.db.ExecContext(ctx, `
 		UPDATE gofire_schema.enqueued_jobs
		SET status = 'succeeded',
		    executed_at = NOW(),
		    finished_at = NOW()
		WHERE id = $1
	`, jobID)
}

func (r *EnqueuedJobRepository) MarkFailure(ctx context.Context, jobID int, errMsg string, attempts int, maxAttempts int) {
	status := state.StatusFailed
	if attempts+1 >= constants.MaxRetryAttempt {
		status = state.StatusDead
	}
	r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET attempts = attempts + 1,
		    last_error = $2,
		    status = $3
		WHERE id = $1
	`, jobID, errMsg, status)
}

func (r *EnqueuedJobRepository) mapSqlRowsToJob(rows *sql.Rows) (*task.EnqueuedJob, error) {
	var job task.EnqueuedJob
	if err := rows.Scan(
		&job.ID,
		&job.Name,
		&job.Payload,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.ScheduledAt,
		&job.ExecutedAt,
		&job.FinishedAt,
		&job.LastError,
		&job.LockedBy,
		&job.LockedAt,
		&job.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &job, nil
}
