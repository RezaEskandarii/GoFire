package mocks

import (
	"context"
	"database/sql"
	"errors"
	"gofire/internal/models"
	"gofire/internal/state"
	"sync"
	"time"
)

type MockCronJobRepository struct {
	mu     sync.Mutex
	jobs   map[int64]*models.CronJob
	nextID int64
}

func NewMockCronJobRepository() *MockCronJobRepository {
	return &MockCronJobRepository{
		jobs: make(map[int64]*models.CronJob),
	}
}

func (m *MockCronJobRepository) AddOrUpdate(ctx context.Context, jobName string, nextRunAt time.Time, expression string, args ...any) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, job := range m.jobs {
		if job.Name == jobName {
			job.NextRunAt = nextRunAt
			job.Expression = expression
			return job.ID, nil
		}
	}

	m.nextID++
	job := &models.CronJob{
		ID:         m.nextID,
		Name:       jobName,
		NextRunAt:  nextRunAt,
		Expression: expression,
		Status:     state.StatusQueued,
	}
	m.jobs[job.ID] = job
	return job.ID, nil
}

func (m *MockCronJobRepository) FetchDueCronJobs(ctx context.Context, page, pageSize int) (*models.PaginationResult[models.CronJob], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var due []models.CronJob
	now := time.Now()

	for _, job := range m.jobs {
		if !job.NextRunAt.After(now) && job.LockedBy == nil {
			due = append(due, *job)
		}
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(due) {
		start = len(due)
	}
	if end > len(due) {
		end = len(due)
	}

	return &models.PaginationResult[models.CronJob]{
		Items:      due[start:end],
		TotalItems: len(due),
		Page:       page,
	}, nil
}

func (m *MockCronJobRepository) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists || job.LockedBy != nil {
		return false, nil
	}
	now := time.Now()
	job.LockedBy = &lockedBy
	job.LockedAt = &now
	return true, nil
}

func (m *MockCronJobRepository) UnLockJob(ctx context.Context, jobID int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists || job.LockedBy == nil {
		return false, nil
	}
	job.LockedBy = nil
	job.LockedAt = nil
	return true, nil
}

func (m *MockCronJobRepository) GetAll(ctx context.Context, page, pageSize int, status state.JobStatus) (*models.PaginationResult[models.CronJob], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []models.CronJob
	for _, job := range m.jobs {
		if job.Status == status {
			list = append(list, *job)
		}
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(list) {
		start = len(list)
	}
	if end > len(list) {
		end = len(list)
	}

	return &models.PaginationResult[models.CronJob]{
		Items:      list[start:end],
		TotalItems: len(list),
		Page:       page,
	}, nil
}

func (m *MockCronJobRepository) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	counts := make(map[state.JobStatus]int)
	for _, job := range m.jobs {
		counts[job.Status]++
	}
	return counts, nil
}

func (m *MockCronJobRepository) UpdateJobRunTimes(ctx context.Context, jobID int64, lastRunAt, nextRunAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.LastRunAt = &lastRunAt
	job.NextRunAt = nextRunAt
	return nil
}

func (m *MockCronJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.Status = state.StatusSucceeded
	job.LockedBy = nil
	job.LockedAt = nil
	return nil
}

func (m *MockCronJobRepository) MarkFailure(ctx context.Context, jobID int64, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.Status = state.StatusFailed
	job.LastError = sql.NullString{
		String: errMsg,
		Valid:  true,
	}
	job.LockedBy = nil
	job.LockedAt = nil
	return nil
}

func (m *MockCronJobRepository) Activate(ctx context.Context, jobID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.IsActive = true
	return nil
}

func (m *MockCronJobRepository) DeActivate(ctx context.Context, jobID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	job.IsActive = false
	return nil
}

func (m *MockCronJobRepository) Close() error {
	return nil
}
