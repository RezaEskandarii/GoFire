package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gofire/internal/models"
	"math"
	"time"
)

type PostgresCronJobRepository struct {
	db *sql.DB
}

func NewCronJobRepository(db *sql.DB) PostgresCronJobRepository {
	return PostgresCronJobRepository{db: db}
}

func (r *PostgresCronJobRepository) AddOrUpdate(ctx context.Context, jobName string, scheduledAt time.Time, args []interface{}, expression string) (int64, error) {
	query := `
		INSERT INTO gofire_schema.cron_jobs (name, next_run_at, payload,expression,created_at,updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
		ON CONFLICT (name) DO UPDATE SET
			next_run_at = $2,
			payload = $3,
			updated_at = now(),
			expression $4
		RETURNING id
	`

	// Convert args to JSON
	payloadJSON, err := json.Marshal(args)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal payload: %w", err)
	}

	var jobID int64
	err = r.db.QueryRowContext(ctx, query, jobName, scheduledAt, payloadJSON, expression).Scan(&jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert or update cron job: %w", err)
	}

	return jobID, nil
}

func (r *PostgresCronJobRepository) FetchDueCronJobs(ctx context.Context, page int, pageSize int) (*models.PaginationResult[models.CronJob], error) {

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	// WHERE clause
	where := "is_active = TRUE AND (next_run_at IS NULL OR next_run_at <= now())"
	var args []interface{}
	argIndex := 2

	countQuery := `SELECT COUNT(*) FROM gofire_schema.cron_jobs WHERE ` + where
	selectQuery := `
		SELECT id, name, payload, status, last_error,
		       locked_by, locked_at, created_at,
		       last_run_at, next_run_at, is_active, expression
		FROM gofire_schema.cron_jobs
		WHERE ` + where + fmt.Sprintf(" ORDER BY next_run_at ASC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	// Count total items
	var totalItems int
	err := r.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&totalItems)
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

func (r *PostgresCronJobRepository) UpdateJobRunTimes(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET last_run_at = $1, next_run_at = $2
	WHERE id = $3;
	`
	_, err := r.db.ExecContext(ctx, query, lastRunAt, nextRunAt, jobID)
	return err
}

func (r *PostgresCronJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET status = 'success', last_error = NULL
	WHERE id = $1;
	`
	_, err := r.db.ExecContext(ctx, query, jobID)
	return err
}

func (r *PostgresCronJobRepository) MarkFailure(ctx context.Context, jobID int64, errMsg string) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET status = 'failed', last_error = $1
	WHERE id = $2;
	`
	_, err := r.db.ExecContext(ctx, query, errMsg, jobID)
	return err
}

func (r *PostgresCronJobRepository) Activate(ctx context.Context, jobID int64) error {
	return r.executeIsActivateQuery(ctx, jobID, true)
}

func (r *PostgresCronJobRepository) DeActivate(ctx context.Context, jobID int64) error {
	return r.executeIsActivateQuery(ctx, jobID, false)
}

func (r *PostgresCronJobRepository) executeIsActivateQuery(ctx context.Context, jobID int64, isActive bool) error {
	query := `
	UPDATE gofire_schema.cron_jobs
	SET is_active = $1
	WHERE id = $2;
	`
	_, err := r.db.ExecContext(ctx, query, isActive, jobID)
	return err
}
