-- +goose Up
CREATE TABLE suppliers (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    name           TEXT        NOT NULL,
    email          TEXT        NOT NULL DEFAULT '',
    phone          TEXT        NOT NULL DEFAULT '',
    country        TEXT        NOT NULL DEFAULT '',
    lead_time_days INTEGER,
    status         TEXT        NOT NULL DEFAULT 'active',
    notes          TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT suppliers_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX suppliers_tenant_id_idx ON suppliers (tenant_id);

ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE suppliers FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON suppliers
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE suppliers;
