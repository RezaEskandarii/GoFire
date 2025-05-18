package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/models"
	"gofire/internal/state"
	"math"
	"time"
)

type PostgresEnqueuedJobRepository struct {
	db *sql.DB
}

func NewPostgresEnqueuedJobRepository(db *sql.DB) *PostgresEnqueuedJobRepository {
	return &PostgresEnqueuedJobRepository{
		db: db,
	}
}

func (r *PostgresEnqueuedJobRepository) Insert(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}) (int64, error) {

	payloadJSON, err := json.Marshal(args)
	if err != nil {
		return -1, err
	}

	query := `
        INSERT INTO gofire_schema.enqueued_jobs (
            name,
            payload, 
            scheduled_at, 
            max_attempts,
            created_at
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
func (r *PostgresEnqueuedJobRepository) FetchDueJobs(
	ctx context.Context,
	page int,
	pageSize int,
	status *state.JobStatus,
	scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error) {

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

	var jobs []models.EnqueuedJob
	for rows.Next() {
		job, err := r.mapSqlRowsToJob(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, *job)
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	result := &models.PaginationResult[models.EnqueuedJob]{
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

func (r *PostgresEnqueuedJobRepository) LockJob(ctx context.Context, job *models.EnqueuedJob, lockedBy string) (bool, error) {
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

func (r *PostgresEnqueuedJobRepository) MarkSuccess(ctx context.Context, jobID int) error {
	_, err := r.db.ExecContext(ctx, `
 		UPDATE gofire_schema.enqueued_jobs
		SET status = 'succeeded',
		    executed_at = NOW(),
		    finished_at = NOW()
		WHERE id = $1
	`, jobID)

	return err
}

func (r *PostgresEnqueuedJobRepository) MarkFailure(ctx context.Context, jobID int, errMsg string, attempts int, maxAttempts int) error {
	status := state.StatusFailed
	if attempts+1 >= constants.MaxRetryAttempt {
		status = state.StatusDead
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET attempts = attempts + 1,
		    last_error = $2,
		    status = $3
		WHERE id = $1
	`, jobID, errMsg, status)

	return err
}

func (r *PostgresEnqueuedJobRepository) UnlockStaleJobs(ctx context.Context, timeout time.Duration) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE gofire_schema.enqueued_jobs
        SET status = $1, 
            locked_by = NULL,
            locked_at = NULL
        WHERE status = $2 AND locked_at <= NOW()
    `,
		state.StatusQueued,
		state.StatusProcessing)
	return err
}

func (r *PostgresEnqueuedJobRepository) mapSqlRowsToJob(rows *sql.Rows) (*models.EnqueuedJob, error) {
	var job models.EnqueuedJob
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
