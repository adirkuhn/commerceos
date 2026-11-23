-- +goose Up
-- Admin AI assistant (domain/assistant). assistant_requests is both the
-- conversation header and the durable work queue: the scheduler's worker
-- locks due rows (status = 'queued' AND next_attempt_at <= now) with
-- FOR UPDATE SKIP LOCKED. A request never leaves the queue on its own;
-- only a user cancels it.
CREATE TABLE assistant_requests (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    user_id         TEXT        NOT NULL,
    user_role       TEXT        NOT NULL,
    title           TEXT        NOT NULL,
    status          TEXT        NOT NULL,
    attempts        INT         NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    wait_reason     TEXT        NOT NULL DEFAULT '',
    last_error      TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_requests_status_check
        CHECK (status IN ('queued', 'processing', 'needs_input', 'answered', 'cancelled')),
    CONSTRAINT assistant_requests_wait_reason_check
        CHECK (wait_reason IN ('', 'model_busy', 'no_provider', 'provider_unsupported', 'model_error'))
);

CREATE INDEX assistant_requests_tenant_created_idx ON assistant_requests (tenant_id, created_at DESC, id DESC);
-- The worker's poll: only queued rows, cheapest first.
CREATE INDEX assistant_requests_due_idx ON assistant_requests (next_attempt_at) WHERE status = 'queued';

ALTER TABLE assistant_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE assistant_requests FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON assistant_requests
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- One row per conversation entry, in seq order. tool_calls holds the
-- model's calls on an assistant message; tool messages answer one call.
CREATE TABLE assistant_messages (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    request_id   UUID        NOT NULL REFERENCES assistant_requests (id) ON DELETE CASCADE,
    seq          INT         NOT NULL,
    role         TEXT        NOT NULL,
    content      TEXT        NOT NULL DEFAULT '',
    tool_calls   JSONB       NOT NULL DEFAULT '[]',
    tool_call_id TEXT        NOT NULL DEFAULT '',
    tool_name    TEXT        NOT NULL DEFAULT '',
    question     BOOLEAN     NOT NULL DEFAULT FALSE,
    notice       TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_messages_role_check CHECK (role IN ('user', 'assistant', 'tool')),
    CONSTRAINT assistant_messages_request_seq_key UNIQUE (request_id, seq)
);

CREATE INDEX assistant_messages_tenant_id_idx ON assistant_messages (tenant_id);

ALTER TABLE assistant_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE assistant_messages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON assistant_messages
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);


-- +goose Down
DROP TABLE assistant_messages;
DROP TABLE assistant_requests;
