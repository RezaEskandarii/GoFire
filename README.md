# GoFire - Job Scheduler

GoFire is a powerful distributed job scheduling system written in Go. It provides a robust solution for managing both one-time and recurring background tasks with support for distributed execution, job monitoring, and a web dashboard.

## Features

- **Distributed Job Processing**: Run jobs across multiple instances with distributed locking
- **Multiple Storage Backends**: Support for both PostgreSQL and Redis
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
go get github.com/yourusername/gofire
```

## Quick Start

Here's a basic example of how to use GoFire:

```go
package main

import (
  "context"
  "gofire/internal/gofire"
  "gofire/internal/models/config"
  "log"
  "time"
)

func main() {
  // Configure GoFire
  cfg, err := config.NewGofireConfig("instance-name",
    config.WithEnqueueInterval(60),  // Enqueue check interval in seconds
    config.WithScheduleInterval(60), // Scheduler check interval in seconds
    config.WithWorkerCount(15),      // Number of concurrent workers
    config.WithBatchSize(500),       // Batch size for job processing
    config.WithPostgresConfig(config.PostgresConfig{
      ConnectionUrl: "postgres://user:pass@localhost:5432/dbname",
    }),
    config.WithAdminDashboardConfig("admin", "password", "secret-key", 8080),
  )

  if err != nil {
    log.Fatal(err)
  }

  // Register job handlers
  _ = cfg.RegisterHandler(config.MethodHandler{
    JobName: "SendSMS",
    Func: func(args ...any) error {
      to := args[0].(string)
      message := args[1].(string)
      return sendSms(to, message)
    },
  })

  _ = cfg.RegisterHandler(config.MethodHandler{
    JobName: "DailyEmailNotificationJob",
    Func: func(args ...any) error {
      address := args[0].(string)
      body := args[1].(string)
      return sendEmail(address, body)
    },
  })

  // Initialize GoFire
  jobManager, err := gofire.BootJobManager(context.Background(), *cfg)
  if err != nil {
    log.Fatal(err)
  }
  defer jobManager.GracefulExit() // Graceful shutdown to release system resources and close database connections

  // Enqueue a job
  jobManager.Enqueue(ctx, "SendSMS", time.Now().Add(time.Minute*20), "123456789", "Hello!")

  // Schedule a job
  jobManager.Schedule(context.Background(), "DailyEmailNotificationJob", "30 8 * * *", "alice@example.com", "Hello!")
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

#### Redis
```go
cfg.WithRedisConfig(config.RedisConfig{
    Address:  "localhost:6379",
    Password: "password",
    DB:       0,
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
jobManager, err := gofire.SetUp(context.Background(), *cfg)
if err != nil {
    log.Fatal(err)
}
defer jobManager.ShutDown() // Ensures proper cleanup of resources

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
