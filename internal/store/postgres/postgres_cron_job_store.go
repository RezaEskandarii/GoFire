package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gofire/internal/models"
	"gofire/internal/state"
	"log"
	"math"
	"time"
)

type PostgresCronJobStore struct {
	db *sql.DB
}

func NewPostgresCronJobStore(db *sql.DB) *PostgresCronJobStore {
	return &PostgresCronJobStore{db: db}
}

func (r *PostgresCronJobStore) AddOrUpdate(ctx context.Context, jobName string, scheduledAt time.Time, expression string, args ...any) (int64, error) {

	query := `
		INSERT INTO gofire_schema.cron_jobs (name, next_run_at, payload,expression,created_at,updated_at,status)
		VALUES ($1, $2, $3, $4, now(), now(),$5)
		ON CONFLICT (name) DO UPDATE SET
			next_run_at = $2,
			payload = $3,
			updated_at = now(),
			expression = $4
		RETURNING id
	`

	// Convert args to JSON
	payloadJSON, err := json.Marshal(args)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	var jobID int64
	err = r.db.QueryRowContext(ctx, query, jobName, scheduledAt, payloadJSON, expression, state.StatusQueued).Scan(&jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert or update cron job: %w", err)
	}

	return jobID, nil
}

func (r *PostgresCronJobStore) FetchDueCronJobs(ctx context.Context, page int, pageSize int) (*models.PaginationResult[models.CronJob], error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	// TTL for job lock expiration.
	// Jobs in 'processing' status will be considered "due" again if locked_at is older than this.
	// WARNING: In multi-node / multi-pod environments, this logic can lead to duplicate job execution
	// if there's no distributed lock mechanism in place.
	// It's strongly recommended to use a distributed lock (e.g., Redis, PostgreSQL advisory locks, etc.)
	// along with this TTL to ensure job safety across nodes.
	lockTTL := "60 minutes"

	where := `
		is_active = TRUE
		AND (next_run_at IS NULL OR next_run_at <= now())
		AND (
			status = 'queued'
			OR status = 'retrying'
			OR (status = 'processing' AND locked_at < now() - interval '` + lockTTL + `')
		)
	`

	var args []interface{}
	argIndex := 1

	countQuery := `SELECT COUNT(*) FROM gofire_schema.cron_jobs WHERE ` + where
	selectQuery := `
		SELECT id, name, payload, status, last_error,
		       locked_by, locked_at, created_at,
		       last_run_at, next_run_at, is_active, expression
		FROM gofire_schema.cron_jobs
		WHERE ` + where + fmt.Sprintf(" ORDER BY next_run_at ASC NULLS FIRST LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	// Count total items
	var totalItems int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	// Query paged results
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.CronJob
	for rows.Next() {
		var job models.CronJob
		err := rows.Scan(
			&job.ID, &job.Name, &job.Payload, &job.Status, &job.LastError,
			&job.LockedBy, &job.LockedAt, &job.CreatedAt,
			&job.LastRunAt, &job.NextRunAt, &job.IsActive, &job.Expression,
		)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		jobs = append(jobs, job)
	}

	// Pagination metadata
	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	result := &models.PaginationResult[models.CronJob]{
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

func (r *PostgresCronJobStore) GetAll(ctx context.Context, page int, pageSize int, status state.JobStatus) (*models.PaginationResult[models.CronJob], error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var args []interface{}
	where := "TRUE"

	argIndex := 1
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	countQuery := `SELECT COUNT(*) FROM gofire_schema.cron_jobs WHERE ` + where
	selectQuery := fmt.Sprintf(`
		SELECT id, name, payload, status, last_error,
		       locked_by, locked_at, created_at,
		       last_run_at, next_run_at, is_active, expression
		FROM gofire_schema.cron_jobs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	// Count total items
	var totalItems int
	err := r.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	// Fetch rows
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.CronJob
	for rows.Next() {
		var job models.CronJob
		err := rows.Scan(
			&job.ID, &job.Name, &job.Payload, &job.Status, &job.LastError,
			&job.LockedBy, &job.LockedAt, &job.CreatedAt,
			&job.LastRunAt, &job.NextRunAt, &job.IsActive, &job.Expression,
		)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		jobs = append(jobs, job)
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	result := &models.PaginationResult[models.CronJob]{
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
func (r *PostgresCronJobStore) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) AS count
		FROM gofire_schema.cron_jobs
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

func (r *PostgresCronJobStore) UpdateJobRunTimes(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET last_run_at = $1, next_run_at = $2
	WHERE id = $3;
	`
	_, err := r.db.ExecContext(ctx, query, lastRunAt, nextRunAt, jobID)
	return err
}

func (r *PostgresCronJobStore) MarkSuccess(ctx context.Context, jobID int64) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET status = $1,
	        last_error = NULL,
	        SET locked_at = NULL,
            locked_by = NULL
	WHERE id = $2;
	`
	_, err := r.db.ExecContext(ctx, query, state.StatusSucceeded, jobID)
	return err
}

func (r *PostgresCronJobStore) MarkFailure(ctx context.Context, jobID int64, errMsg string) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET status = 'failed', last_error = $1
	WHERE id = $2;
	`
	_, err := r.db.ExecContext(ctx, query, errMsg, jobID)
	return err
}

func (r *PostgresCronJobStore) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {

	res, err := r.db.ExecContext(ctx, `
		UPDATE gofire_schema.cron_jobs
		SET locked_at = NOW(), 
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

func (r *PostgresCronJobStore) UnLockJob(ctx context.Context, jobID int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
        UPDATE gofire_schema.cron_jobs
        SET locked_at = NULL,
            locked_by = NULL
        WHERE id = $1
    `, jobID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (r *PostgresCronJobStore) Activate(ctx context.Context, jobID int64) error {
	return r.executeChangeActivateQuery(ctx, jobID, true)
}

func (r *PostgresCronJobStore) DeActivate(ctx context.Context, jobID int64) error {
	return r.executeChangeActivateQuery(ctx, jobID, false)
}

func (r *PostgresCronJobStore) executeChangeActivateQuery(ctx context.Context, jobID int64, isActive bool) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET is_active = $1
	WHERE id = $2;
	`
	_, err := r.db.ExecContext(ctx, query, isActive, jobID)
	return err
}

func (r *PostgresCronJobStore) Close() error {
	return r.db.Close()
}
