CREATE TABLE recap_import_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    recap_id BIGINT NOT NULL UNIQUE
        REFERENCES recaps(id)
        ON DELETE CASCADE,

    file_path TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'queued',

    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    CONSTRAINT recap_import_jobs_status_check
        CHECK (
            status IN (
                'queued',
                'processing',
                'completed',
                'failed'
            )
        ),

    CONSTRAINT recap_import_jobs_attempts_check
        CHECK (attempts >= 0)
);

CREATE INDEX recap_import_jobs_queue_idx
    ON recap_import_jobs (created_at, id)
    WHERE status = 'queued';