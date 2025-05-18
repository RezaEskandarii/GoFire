package app

import (
	"context"
	"encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/db"
	"gofire/internal/models"
	"gofire/internal/state"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
	"time"
)

type EnqueueScheduler struct {
	repository db.EnqueuedJobRepository
	instance   string
	lock       db.Lock
	handlers   map[string]func([]interface{}) error
}

func NewScheduler(repository db.EnqueuedJobRepository, lock db.Lock, instance string) EnqueueScheduler {
	return EnqueueScheduler{
		repository: repository,
		instance:   instance,
		lock:       lock,
		handlers:   make(map[string]func([]interface{}) error),
	}
}

func (s *EnqueueScheduler) RegisterHandler(name string, fn func([]interface{}) error) {
	s.handlers[name] = fn
}

func (s *EnqueueScheduler) GetHandler(name string) (func([]interface{}) error, error) {
	fn, ok := s.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return fn, nil
}

func (s *EnqueueScheduler) Enqueue(ctx context.Context, name string, scheduledAt time.Time, args []interface{}) (int64, error) {
	return s.repository.Insert(ctx, name, scheduledAt, args)
}

func (s *EnqueueScheduler) ProcessEnqueues(ctx context.Context) error {
	enqueueLock := constants.EnqueueLock
	if err := s.lock.AcquirePostgresDistributedLock(enqueueLock); err != nil {
		return err
	}
	defer s.lock.ReleasePostgresDistributedLock(enqueueLock)

	if err := s.repository.UnlockStaleJobs(ctx, time.Minute*5); err != nil {
		log.Fatal(err.Error())
	}

	sem := semaphore.NewWeighted(5)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
			now := time.Now()
			status := state.StatusQueued
			jobsList, err := s.repository.FetchDueJobs(ctx, 1, 100, &status, &now)
			if err != nil {
				log.Println("fetch error:", err)
				time.Sleep(2 * time.Second)
				continue
			}
			jobs := jobsList.Items

			for _, job := range jobs {
				ok, err := s.repository.LockJob(ctx, &job, s.instance)
				if err != nil || !ok {
					continue
				}

				if err := sem.Acquire(ctx, 1); err != nil {
					break
				}
				wg.Add(1)

				go func(job models.EnqueuedJob) {
					defer sem.Release(1)
					defer wg.Done()

					handler, err := s.GetHandler(job.Name)
					if err != nil {
						log.Println(err.Error())
					}
					var args []interface{}
					_ = json.Unmarshal(job.Payload, &args)

					if err := handler(args); err != nil && state.IsValidTransition(job.Status, state.StatusFailed) {

						s.repository.MarkFailure(ctx, job.ID, err.Error(), job.Attempts+1, job.MaxAttempts)

					} else if err == nil && state.IsValidTransition(job.Status, state.StatusSucceeded) {
						s.repository.MarkSuccess(ctx, job.ID)
					}

				}(job)
			}

			time.Sleep(1 * time.Second)
		}
	}
}
