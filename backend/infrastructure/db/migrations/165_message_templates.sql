-- +goose Up
CREATE TABLE message_templates (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    name       TEXT        NOT NULL,
    category   TEXT        NOT NULL,
    channels   JSONB       NOT NULL,
    subject    TEXT        NOT NULL DEFAULT '',
    body       TEXT        NOT NULL,
    enabled    BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT message_templates_category_check CHECK (category IN ('purchase', 'order_status', 'stock_info', 'tracking', 'custom'))
);

CREATE INDEX message_templates_tenant_id_idx ON message_templates (tenant_id);

ALTER TABLE message_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_templates FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON message_templates
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE message_templates;
