CREATE TABLE IF NOT EXISTS gofire_schema.enqueued_jobs
(
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    scheduled_at TIMESTAMPTZ NOT NULL,
    executed_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    last_error TEXT,
    locked_by TEXT,
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_scheduled_at
    ON gofire_schema.enqueued_jobs (status, scheduled_at);

CREATE INDEX IF NOT EXISTS idx_jobs_scheduled_at
    ON gofire_schema.enqueued_jobs (scheduled_at);
