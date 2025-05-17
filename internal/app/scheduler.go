package app

import (
	"context"
	"fmt"
	"gofire/internal/db"
	"time"
)

type Scheduler struct {
	repository db.JobRepository
	instance   string
	handlers   map[string]func(...any) error
}

func NewScheduler(repository db.JobRepository, instance string) Scheduler {
	return Scheduler{
		repository: repository,
		instance:   instance,
		handlers:   make(map[string]func(...any) error),
	}
}

func (s *Scheduler) RegisterHandler(name string, fn func(...any) error) {
	s.handlers[name] = fn
}

func (s *Scheduler) GetHandler(name string) (func(...any) error, error) {
	fn, ok := s.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return fn, nil
}

func (s *Scheduler) Enqueue(ctx context.Context, name string, scheduledAt time.Time, args ...any) (int64, error) {
	return s.repository.InsertEnqueuedJob(ctx, name, scheduledAt, args)
}
