CREATE TABLE IF NOT EXISTS gofire_schema.cron_jobs
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL unique,
    status      VARCHAR(50),
    payload     JSONB        NOT NULL,
    last_error  TEXT,
    locked_by   TEXT,
    locked_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    last_run_at TIMESTAMP,
    finished_at TIMESTAMP,
    next_run_at TIMESTAMP,
    is_active   BOOLEAN               DEFAULT TRUE,
    expression  VARCHAR(50)
);

CREATE INDEX IF NOT EXISTS idx_jobs_status
    ON gofire_schema.cron_jobs USING btree (status);

CREATE INDEX IF NOT EXISTS idx_jobs_next_run_at
    ON gofire_schema.cron_jobs USING btree (next_run_at);
