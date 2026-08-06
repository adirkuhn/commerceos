-- +goose Up
CREATE TABLE ai_connections (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    provider          TEXT        NOT NULL,
    api_key_encrypted TEXT,
    base_url          TEXT,
    model             TEXT        NOT NULL DEFAULT '',
    status            TEXT        NOT NULL DEFAULT 'connected',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_connections_provider_check CHECK (provider IN ('openai', 'anthropic', 'gemini', 'ollama')),
    CONSTRAINT ai_connections_status_check CHECK (status IN ('connected', 'error')),
    CONSTRAINT ai_connections_tenant_unique UNIQUE (tenant_id)
);

CREATE INDEX ai_connections_tenant_id_idx ON ai_connections (tenant_id);

ALTER TABLE ai_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_connections FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON ai_connections
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE ai_connections;
