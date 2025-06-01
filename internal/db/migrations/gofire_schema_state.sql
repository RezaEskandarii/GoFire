CREATE TABLE IF NOT EXISTS gofire_schema_state
(
    id         SERIAL PRIMARY KEY,
    job_id     bigint                            NOT NULL,
    job_status text COLLATE pg_catalog."default" NOT NULL,
    reason     text COLLATE pg_catalog."default",
    created_at timestamp without time zone       NOT NULL,
    data       text COLLATE pg_catalog."default"
);

CREATE INDEX IF NOT EXISTS gofire_schema_state
    ON gofire_schema_state USING btree
        (job_id ASC NULLS LAST);