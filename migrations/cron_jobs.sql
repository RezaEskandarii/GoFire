CREATE TABLE IF NOT EXISTS gofire_schema.cron_jobs
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    payload     JSONB        NOT NULL,
    last_error  TEXT,
    locked_by   TEXT,
    locked_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    is_active   BOOLEAN               DEFAULT TRUE,
    expression  VARCHAR(50)
);

CREATE INDEX IF NOT EXISTS idx_jobs_status
    ON gofire_schema.cron_jobs USING btree (status);

CREATE INDEX IF NOT EXISTS idx_jobs_next_run_at
    ON gofire_schema.cron_jobs USING btree (next_run_at);

ALTER TABLE gofire_schema.cron_jobs
    ADD CONSTRAINT unique_cron_job_name UNIQUE (name);
