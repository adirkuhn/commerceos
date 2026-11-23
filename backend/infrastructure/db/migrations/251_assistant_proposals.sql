-- +goose Up
-- Changes the assistant proposed (domain/assistant.Proposal): nothing is
-- written until an admin confirms. payload is the exact input the action
-- runs with; preview is what the admin is shown.
CREATE TABLE assistant_proposals (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    request_id  UUID        NOT NULL REFERENCES assistant_requests (id) ON DELETE CASCADE,
    kind        TEXT        NOT NULL,
    status      TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    preview     JSONB       NOT NULL,
    last_error  TEXT        NOT NULL DEFAULT '',
    result_id   TEXT        NOT NULL DEFAULT '',
    result_ref  TEXT        NOT NULL DEFAULT '',
    decided_by  TEXT        NOT NULL DEFAULT '',
    decided_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assistant_proposals_kind_check CHECK (kind IN ('create_quote', 'convert_quote')),
    CONSTRAINT assistant_proposals_status_check
        CHECK (status IN ('pending', 'executing', 'confirmed', 'rejected', 'superseded'))
);

CREATE INDEX assistant_proposals_request_idx ON assistant_proposals (request_id, created_at);
CREATE INDEX assistant_proposals_tenant_id_idx ON assistant_proposals (tenant_id);

ALTER TABLE assistant_proposals ENABLE ROW LEVEL SECURITY;
ALTER TABLE assistant_proposals FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON assistant_proposals
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- A request can now wait on a proposal's decision.
ALTER TABLE assistant_requests DROP CONSTRAINT assistant_requests_status_check;
ALTER TABLE assistant_requests ADD CONSTRAINT assistant_requests_status_check
    CHECK (status IN ('queued', 'processing', 'needs_input', 'answered', 'awaiting_confirmation', 'cancelled'));

-- The message that made or decided a proposal points at it.
ALTER TABLE assistant_messages ADD COLUMN proposal_id UUID REFERENCES assistant_proposals (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE assistant_messages DROP COLUMN proposal_id;
UPDATE assistant_requests SET status = 'answered' WHERE status = 'awaiting_confirmation';
ALTER TABLE assistant_requests DROP CONSTRAINT assistant_requests_status_check;
ALTER TABLE assistant_requests ADD CONSTRAINT assistant_requests_status_check
    CHECK (status IN ('queued', 'processing', 'needs_input', 'answered', 'cancelled'));
DROP TABLE assistant_proposals;
