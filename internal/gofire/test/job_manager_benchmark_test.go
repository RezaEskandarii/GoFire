package test

import (
	"context"
	"fmt"
	"gofire/internal/models/config"
	"testing"
	"time"
)

func BenchmarkEnqueue(b *testing.B) {

	b.StartTimer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobName := "job-5"
	enqueueAt := time.Now().Add(3 * time.Second)
	args := []any{"1234567890", "This is a test message"}

	for i := 0; i < b.N; i++ {
		_, err := benchJobManager.Enqueue(ctx, jobName, enqueueAt, args...)
		if err != nil {
			b.Fatalf("enqueue failed at iteration %d: %v", i, err)
		}
	}
}
func BenchmarkEnqueueParallel(b *testing.B) {

	for i := 0; i < 20; i++ {
		jobName := fmt.Sprintf("job-%d", i)
		cfg.RegisterHandler(config.MethodHandler{
			MethodName: jobName,
			Func: func(args ...any) error {
				to := args[0].(string)
				message := args[1].(string)
				return sendSms(to, message, false)
			},
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobName := "job-5"
	enqueueAt := time.Now().Add(3 * time.Second)
	args := []any{"1234567890", "This is a test message"}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := benchJobManager.Enqueue(ctx, jobName, enqueueAt, args...)
			if err != nil {
				b.Errorf("enqueue failed: %v", err)
			}
		}
	})
}
