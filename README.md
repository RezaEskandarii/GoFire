# GoFire - Job Scheduler

GoFire is a powerful distributed job scheduling system written in Go. It provides a robust solution for managing both one-time and recurring background tasks with support for distributed execution, job monitoring, and a web dashboard.

## Features

- **Distributed Job Processing**: Run jobs across multiple instances with distributed locking
- **Storage Backends**: Support for PostgreSQL 
- **Flexible Job Scheduling**:
  - One-time jobs with custom scheduling
  - Cron-based recurring jobs
  - Built-in scheduling helpers (every minute, hour, day, week, month, year)
- **Job Management**:
  - Job status tracking
  - Retry mechanism for failed jobs
  - Job locking and stale job recovery
  - Manual job execution
- **Web Dashboard**: Monitor and control jobs through a web interface
- **Authentication**: Secure dashboard access with user authentication

## Installation

```bash
go get github.com/RezaEskandarii/gofire
```

## Quick Start

Here's a basic example of how to use GoFire:

```go

package main

import (
  "context"
  "crypto/rand"
  "fmt"
  "github.com/RezaEskandarii/gofire/jobmanager"
  "github.com/RezaEskandarii/gofire/types/config"
  "log"
  "math/big"
  "time"
)

const (
  SendSMSJob                = "SendSMSJob"
  DailyEmailNotificationJob = "DailyEmailNotificationJob"
)

func main() {
  // Recover from panics
  defer func() {
    if r := recover(); r != nil {
      fmt.Printf("panic: %v\n", r)
    }
  }()

  ctx := context.Background()

  // 1. Configure Database
  dbConfig := config.PostgresConfig{
    ConnectionUrl: "host=127.0.0.1 port=5432 user=postgres password=123456 dbname=gofire sslmode=disable",
  }

  // 2. Configure GoFire
  cfg, err := config.NewGofireConfig("basket-microservice",
    config.WithEnqueueInterval(5),
    config.WithScheduleInterval(5),
    config.WithWorkerCount(1),
    config.WithBatchSize(3000),
    config.WithPostgresConfig(dbConfig),
    config.WithAdminDashboardConfig("admin", "password", "secret-key", 8080),
  )
  fatalOnError(err, "Failed to create GoFire config")

  // 3. Register Job Handlers
  handlers := []config.MethodHandler{
    {
      JobName: SendSMSJob,
      Func: func(args ...any) error {
        to := args[0].(string)
        message := args[1].(string)
        return sendSms(to, message)
      },
    },
    {
      JobName: DailyEmailNotificationJob,
      Func: func(args ...any) error {
        address := args[0].(string)
        body := args[1].(string)
        return sendEmail(address, body)
      },
    },
  }

  // Register all handlers
  if err := cfg.RegisterHandlers(handlers); err != nil {
    log.Fatal(err)
  }

  // 4. Initialize JobManager (single entry point: creates container, starts workers + dashboard)
  jobManager, err := jobmanager.New(ctx, cfg)
  fatalOnError(err, "Failed to initialize job manager")
  defer jobManager.GracefulExit()

  // 5. Enqueue a job
  for i := 0; i < 105; i++ {
    message := fmt.Sprintf("Your Login OTP is: %s", generateOTP())
    number := fmt.Sprintf("12345678%d", i)

    _, err = jobManager.Enqueue(ctx, SendSMSJob, time.Now().Add(time.Duration(i)+time.Second), number, message)
    if err != nil {
      log.Printf("Error enqueuing job: %v", err) // Log error without stopping the program
      return
    }
  }

  // 6. Schedule a cron job
  _, err = jobManager.Schedule(ctx, DailyEmailNotificationJob, "30 8 * * *", "alice@example.com", "Hello!")
  if err != nil {
    log.Printf("Error scheduling job: %v", err) // Log error without stopping the program
  }
}

// fatalOnError is a helper function to handle critical errors
func fatalOnError(err error, msg string) {
  if err != nil {
    log.Fatalf("%s: %v", msg, err)
  }
}

func sendSms(to string, message string) error {
  fmt.Printf("Number: %s, Message: %s\n", to, message)
  return nil
}

func sendEmail(address string, body string) error {
  fmt.Printf("Send Email to : "+address, body)
  return nil
}

func generateOTP() string {
  nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
  if err != nil {
    panic(err)
  }

  code := fmt.Sprintf("%06d", nBig.Int64())
  return code
}

```

## Job Manager API

### One-time Jobs

```go
// Enqueue a job to run at a specific time
jobID, err := jobManager.Enqueue(ctx, "job_name", time.Now().Add(1*time.Hour), arg1, arg2)

// Remove a queued job
err := jobManager.RemoveEnqueue(ctx, jobID)

// Find job details
job, err := jobManager.FindEnqueue(ctx, jobID)
```

### Recurring Jobs

```go
// Schedule a job with a cron expression
jobID, err := jobManager.Schedule(ctx, "job_name", "* * * * *", arg1, arg2)

// Built-in scheduling helpers
jobID, err := jobManager.ScheduleEveryMinute(ctx, "job_name", arg1, arg2)
jobID, err := jobManager.ScheduleEveryHour(ctx, "job_name", arg1, arg2)
jobID, err := jobManager.ScheduleEveryDay(ctx, "job_name", arg1, arg2)
jobID, err := jobManager.ScheduleEveryWeek(ctx, "job_name", arg1, arg2)
jobID, err := jobManager.ScheduleEveryMonth(ctx, "job_name", arg1, arg2)
jobID, err := jobManager.ScheduleEveryYear(ctx, "job_name", arg1, arg2)

// Activate/Deactivate a scheduled job
jobManager.ActivateSchedule(ctx, jobID)
jobManager.DeActivateSchedule(ctx, jobID)
```

### Timer-based Scheduling

```go
// Schedule a job using timers
err := jobManager.ScheduleInvokeWithTimer(ctx, "job_name", "* * * * *", arg1, arg2)

// Schedule a custom function using timers
err := jobManager.ScheduleFuncWithTimer(ctx, "* * * * *", func(args ...any) error {
    // Custom job logic
    return nil
}, arg1, arg2)
```

## Job Statuses

Jobs can have the following statuses:
- `queued`: Job is waiting to be processed
- `processing`: Job is currently being executed
- `succeeded`: Job completed successfully
- `failed`: Job execution failed
- `retrying`: Failed job is scheduled for retry
- `dead`: Job failed permanently after max retries

## Configuration

### Core Settings

```go
cfg := config.NewGofireConfig("instance-name").
    WithEnqueueInterval(700)     // Check for new enqueued jobs every 700 seconds
    WithSchedulerInterval(3)      // Check for scheduled jobs every 3 seconds
    WithWorkerCount(15)          // Number of concurrent workers
    WithBatchSize(500)           // Batch size for job processing
```

### Storage Options

#### PostgreSQL
```go
cfg.WithPostgresConfig(config.PostgresConfig{
    ConnectionUrl: "postgres://user:pass@localhost:5432/dbname",
})
```

### Dashboard Configuration
```go
cfg.WithAdminDashboardConfig(
    "username",     // Dashboard username
    "password",     // Dashboard password
    "secret-key",   // Secret key for authentication
    "8080",         // Admin dashboard port
)
```

### RabbitMQ Configuration (Optional)
```go
cfg.UseRabbitMQueueWriter(true).
WithRabbitMQConfig(config.RabbitMQConfig{
    URL: "amqp://guest:guest@localhost:5672/",
    Exchange:    "gofire_exchange",
    Queue:       "gofire_jobs",
    RoutingKey:  "jobs.enqueue",
})
```

The RabbitMQ configuration is optional but recommended for high-write scenarios. When enabled:
- Jobs are temporarily stored in RabbitMQ when database write operations are heavy
  Jobs are temporarily stored in RabbitMQ and later written to the database in bulk (e.g., 1000 per batch) to optimize database performance and reduce write overhead.
- Helps prevent database overload during high-write periods

## Web Dashboard

The web dashboard provides a comprehensive interface for monitoring and managing your jobs. Here are some key features:

### Login Page
![Login Page](web/static/assets/images/0-login.JPG)
Secure authentication system to protect your dashboard.

### Job Statistics
![Job Statistics](web/static/assets/images/1-charts.JPG)
Real-time charts and statistics showing job execution metrics.

### Enqueued Jobs
![Enqueued Jobs](web/static/assets/images/2-enqueued.JPG)
Monitor and manage one-time scheduled jobs.

### Cron Jobs
![Cron Jobs](web/static/assets/images/3-cron.JPG)
View and control recurring jobs with cron expressions.

Access the dashboard at `http://localhost:8080` (or your configured port).

## Graceful Shutdown

GoFire provides graceful shutdown functionality to ensure proper cleanup of system resources:

```go
// Initialize GoFire
jobManager, err := jobmanager.New(ctx, cfg)
fatalOnError(err, "Failed to initialize job manager")
defer jobManager.GracefulExit()
// The ShutDown method:
// - Stops all job processors
// - Closes database connections
// - Releases distributed locks
// - Cleans up any temporary resources
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
