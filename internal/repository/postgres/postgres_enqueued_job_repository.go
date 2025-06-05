package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/models"
	"gofire/internal/state"
	"math"
	"strings"
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

func (r *PostgresEnqueuedJobRepository) Insert(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {

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
        VALUES ($1, $2, $3, $4,now())
        returning id
    `

	var jobID int64
	err = r.db.QueryRowContext(ctx, query,
		jobName,
		payloadJSON,
		enqueueAt,
		constants.MaxRetryAttempt,
	).Scan(&jobID)

	return jobID, err

}

func (r *PostgresEnqueuedJobRepository) FindByID(ctx context.Context, id int64) (*models.EnqueuedJob, error) {
	query := `
		SELECT id,
		       name,
		       payload,
		       status,
		       attempts,
		       max_attempts,
		       scheduled_at,
		       executed_at,
		       executed_at,
		       last_error,
		       locked_by,
		       locked_at,
		       created_at
		FROM gofire_schema.enqueued_jobs
		WHERE id = $1
	`

	row, err := r.db.QueryContext(ctx, query, id)
	job, err := r.mapSqlRowsToJob(row)
	if err != nil {
		return nil, fmt.Errorf("job with ID %d not found: %w", id, err)
	}
	return job, nil
}

func (r *PostgresEnqueuedJobRepository) RemoveByID(ctx context.Context, jobID int64) error {
	query := `DELETE FROM gofire_schema.enqueued_jobs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job %d: %w", jobID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no job found with id %d", jobID)
	}

	return nil
}

func (r *PostgresEnqueuedJobRepository) FetchDueJobs(
	ctx context.Context,
	page int,
	pageSize int,
	statuses []state.JobStatus,
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
		args = append(args, scheduledBefore)
		argIndex++
	}

	if len(statuses) > 0 {
		placeholders := []string{}
		for _, s := range statuses {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, s)
			argIndex++
		}
		where += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}

	// Apply lock TTL logic for 'processing' jobs
	lockTTL := "60 minutes"
	where += fmt.Sprintf(" AND (status != 'processing' OR locked_at < now() - interval '%s')", lockTTL)

	countQuery := `SELECT COUNT(*) FROM gofire_schema.enqueued_jobs WHERE ` + where
	selectQuery := `
		SELECT id, name, payload, status, attempts, max_attempts,
		       scheduled_at, executed_at, executed_at, last_error,
		       locked_by, locked_at, created_at
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

func (r *PostgresEnqueuedJobRepository) MarkRetryFailedJobs(ctx context.Context) error {
	query := `
		UPDATE gofire_schema.enqueued_jobs
		SET 
			status = $1,
			scheduled_at = NOW() + INTERVAL '1 minute'
		WHERE id IN (
			SELECT id FROM gofire_schema.enqueued_jobs
			WHERE 
				status = $2
				AND attempts < max_attempts
			ORDER BY scheduled_at ASC
		)
	`

	_, err := r.db.ExecContext(ctx, query, state.StatusRetrying, state.StatusFailed)
	return err
}

func (r *PostgresEnqueuedJobRepository) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {

	res, err := r.db.ExecContext(ctx, `
		UPDATE gofire_schema.enqueued_jobs
		SET locked_at = NOW(), 
		    executed_at = NOW(),
		    locked_by = $1,
		    status = $2
		WHERE id = $3 AND (status = $4 OR status = $5)
	`, lockedBy, state.StatusProcessing, jobID, state.StatusQueued, state.StatusRetrying)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (r *PostgresEnqueuedJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	_, err := r.db.ExecContext(ctx, `
 		UPDATE gofire_schema.enqueued_jobs
		SET status = 'succeeded',
		    executed_at = NOW()
		WHERE id = $1
	`, jobID)

	return err
}

func (r *PostgresEnqueuedJobRepository) MarkFailure(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error {
	status := state.StatusFailed
	if attempts >= constants.MaxRetryAttempt {
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

func (r *PostgresEnqueuedJobRepository) CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM gofire_schema.enqueued_jobs
		WHERE status = $1;
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostgresEnqueuedJobRepository) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) AS count
		FROM gofire_schema.enqueued_jobs
		GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[state.JobStatus]int)
	for rows.Next() {
		var status state.JobStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}

	for _, status := range state.AllStatuses {
		if _, ok := result[status]; !ok {
			result[status] = 0
		}
	}

	return result, nil
}

func (r *PostgresEnqueuedJobRepository) Close() error {
	return r.db.Close()
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
