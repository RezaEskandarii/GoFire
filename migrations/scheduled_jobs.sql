CREATE TABLE IF NOT EXISTS gofire_schema.scheduled_jobs
(
    id          SERIAL PRIMARY KEY,
    job_name    TEXT  NOT NULL,
    args        JSONB NOT NULL,
    cron_expr   TEXT  NOT NULL,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP
);
