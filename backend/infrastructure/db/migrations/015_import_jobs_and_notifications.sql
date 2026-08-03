-- +goose Up

CREATE TABLE import_jobs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    connection_id        UUID        NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    default_warehouse_id UUID        NOT NULL,
    default_currency     TEXT        NOT NULL,
    status               TEXT        NOT NULL DEFAULT 'pending',
    imported             INT         NOT NULL DEFAULT 0,
    already_linked       INT         NOT NULL DEFAULT 0,
    skipped_type         INT         NOT NULL DEFAULT 0,
    errors               JSONB       NOT NULL DEFAULT '[]',
    error_message        TEXT        NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ,
    finished_at          TIMESTAMPTZ,
    CONSTRAINT import_jobs_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

CREATE INDEX import_jobs_connection_id_idx ON import_jobs (connection_id);
CREATE INDEX import_jobs_status_idx ON import_jobs (status);

-- At most one active (pending/running) job per connection — the backend's
-- authoritative guard against double-import-clicks spawning duplicate jobs.
-- application/channels.CreateImportJobUseCase checks proactively; this is
-- the backstop for the race window between check and insert.
CREATE UNIQUE INDEX import_jobs_one_active_per_connection
    ON import_jobs (connection_id)
    WHERE status IN ('pending', 'running');

CREATE TABLE notifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    type       TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    message    TEXT        NOT NULL DEFAULT '',
    read       BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX notifications_tenant_created_idx ON notifications (tenant_id, created_at DESC);

ALTER TABLE import_jobs   ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON import_jobs
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON notifications
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down
DROP TABLE notifications;
DROP TABLE import_jobs;
