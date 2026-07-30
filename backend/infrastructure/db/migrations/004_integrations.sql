-- +goose Up

CREATE TABLE integration_connections (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    provider                  TEXT        NOT NULL,
    store_url                 TEXT        NOT NULL,
    consumer_key              TEXT        NOT NULL,
    consumer_secret_encrypted TEXT        NOT NULL,
    status                    TEXT        NOT NULL DEFAULT 'connected',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, provider, store_url)
);

ALTER TABLE integration_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY integration_connections_tenant_isolation ON integration_connections
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down

DROP TABLE IF EXISTS integration_connections;
