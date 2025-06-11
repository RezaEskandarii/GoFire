// mocks/mock_enqueued_job_repository.go
package mocks

import (
	"context"
	"errors"
	"gofire/internal/models"
	"gofire/internal/state"
	"sync"
	"time"
)

type MockEnqueuedJobRepository struct {
	mu    sync.Mutex
	jobs  map[int64]*models.EnqueuedJob
	idSeq int64
}

func NewMockEnqueuedJobRepository() *MockEnqueuedJobRepository {
	return &MockEnqueuedJobRepository{
		jobs: make(map[int64]*models.EnqueuedJob),
	}
}

func (m *MockEnqueuedJobRepository) FindByID(ctx context.Context, id int64) (*models.EnqueuedJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[id]
	if !exists {
		return nil, errors.New("job not found")
	}
	return job, nil
}

func (m *MockEnqueuedJobRepository) RemoveByID(ctx context.Context, jobID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.jobs, jobID)
	return nil
}

func (m *MockEnqueuedJobRepository) Insert(ctx context.Context, jobName string, enqueueAt time.Time, args ...any) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idSeq++
	job := &models.EnqueuedJob{
		ID:          m.idSeq,
		Name:        jobName,
		Status:      state.StatusQueued,
		ScheduledAt: enqueueAt,
	}
	m.jobs[job.ID] = job
	return job.ID, nil
}

func (m *MockEnqueuedJobRepository) FetchDueJobs(ctx context.Context, page, pageSize int, statuses []state.JobStatus, scheduledBefore *time.Time) (*models.PaginationResult[models.EnqueuedJob], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.EnqueuedJob
	statusMap := make(map[state.JobStatus]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}

	for _, job := range m.jobs {
		if statusMap[job.Status] && (scheduledBefore == nil || job.ScheduledAt.Before(*scheduledBefore)) {
			result = append(result, *job)
		}
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(result) {
		start = len(result)
	}
	if end > len(result) {
		end = len(result)
	}

	return &models.PaginationResult[models.EnqueuedJob]{
		Items:      result[start:end],
		TotalItems: len(result),
		Page:       page,
	}, nil
}

func (m *MockEnqueuedJobRepository) LockJob(ctx context.Context, jobID int64, lockedBy string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok || job.LockedBy != nil {
		return false, nil
	}
	now := time.Now()
	job.LockedBy = &lockedBy
	job.LockedAt = &now
	job.Status = state.StatusProcessing
	return true, nil
}

func (m *MockEnqueuedJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if job, ok := m.jobs[jobID]; ok {
		job.Status = state.StatusSucceeded
		job.LockedBy = nil
		job.LockedAt = nil
		job.ExecutedAt = &now
	}
	return nil
}

func (m *MockEnqueuedJobRepository) MarkFailure(ctx context.Context, jobID int64, errMsg string, attempts int, maxAttempts int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.Attempts = attempts
		job.MaxAttempts = maxAttempts
		job.LastError = &errMsg
		job.LockedBy = nil
		job.LockedAt = nil
		if attempts >= maxAttempts {
			job.Status = state.StatusFailed
		} else {
			job.Status = state.StatusRetrying
		}
	}
	return nil
}

func (m *MockEnqueuedJobRepository) UnlockStaleJobs(ctx context.Context, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, job := range m.jobs {
		if job.LockedAt != nil && now.Sub(*job.LockedAt) > timeout {
			job.LockedBy = nil
			job.LockedAt = nil
			job.Status = state.StatusQueued
		}
	}
	return nil
}

func (m *MockEnqueuedJobRepository) CountJobsByStatus(ctx context.Context, status state.JobStatus) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, job := range m.jobs {
		if job.Status == status {
			count++
		}
	}
	return count, nil
}

func (m *MockEnqueuedJobRepository) CountAllJobsGroupedByStatus(ctx context.Context) (map[state.JobStatus]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	counts := make(map[state.JobStatus]int)
	for _, job := range m.jobs {
		counts[job.Status]++
	}
	return counts, nil
}

func (m *MockEnqueuedJobRepository) MarkRetryFailedJobs(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, job := range m.jobs {
		if job.Status == state.StatusProcessing && job.Attempts < job.MaxAttempts {
			job.Status = state.StatusRetrying
		}
	}
	return nil
}

func (m *MockEnqueuedJobRepository) BulkInsert(ctx context.Context, batch []models.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, job := range batch {
		m.idSeq++
		newJob := models.EnqueuedJob{
			ID:          job.ID,
			ScheduledAt: job.ScheduledAt,
			Name:        job.Name,
		}
		newJob.ID = m.idSeq
		m.jobs[newJob.ID] = &newJob
	}
	return nil
}

func (m *MockEnqueuedJobRepository) Close() error {
	return nil
}
