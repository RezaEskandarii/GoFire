package app

// go get golang.org/x/sync/semaphore
import (
	"context"
	"encoding/json"
	"fmt"
	"gofire/internal/constants"
	"gofire/internal/db"
	"gofire/internal/task"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
	"time"
)

type Scheduler struct {
	repository db.JobRepository
	instance   string
	lock       db.Lock
	handlers   map[string]func([]interface{}) error
}

func NewScheduler(repository db.JobRepository, lock db.Lock, instance string) Scheduler {
	return Scheduler{
		repository: repository,
		instance:   instance,
		lock:       lock,
		handlers:   make(map[string]func([]interface{}) error),
	}
}

func (s *Scheduler) RegisterHandler(name string, fn func([]interface{}) error) {
	s.handlers[name] = fn
}

func (s *Scheduler) GetHandler(name string) (func([]interface{}) error, error) {
	fn, ok := s.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return fn, nil
}

func (s *Scheduler) Enqueue(ctx context.Context, name string, scheduledAt time.Time, args []interface{}) (int64, error) {
	return s.repository.Insert(ctx, name, scheduledAt, args)
}

func (s *Scheduler) Schedule(ctx context.Context, name string, cron string, args ...any) (int64, error) {
	return 0, nil
}

func (s *Scheduler) ProcessEnqueues(ctx context.Context) error {
	enqueueLock := constants.EnqueueLock
	if err := s.lock.AcquirePostgresDistributedLock(enqueueLock); err != nil {
		return err
	}
	defer s.lock.ReleasePostgresDistributedLock(enqueueLock)

	sem := semaphore.NewWeighted(5)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
			jobs, err := s.repository.FetchDueJobs(ctx, 100)
			if err != nil {
				log.Println("fetch error:", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for _, job := range jobs {
				ok, err := s.repository.LockJob(ctx, job.ID, s.instance)
				if err != nil || !ok {
					continue
				}

				if err := sem.Acquire(ctx, 1); err != nil {
					break
				}
				wg.Add(1)

				go func(job task.EnqueuedJob) {
					defer sem.Release(1)
					defer wg.Done()

					handler, err := s.GetHandler(job.Name)
					if err != nil {
						log.Println(err.Error())
					}
					var args []interface{}
					_ = json.Unmarshal(job.Payload, &args)

					if err := handler(args); err != nil {
						s.repository.MarkFailure(ctx, job.ID, err.Error(), job.Attempts, job.MaxAttempts)
					} else {
						s.repository.MarkSuccess(ctx, job.ID)
					}
				}(job)
			}

			time.Sleep(5 * time.Second)
		}
	}
}
