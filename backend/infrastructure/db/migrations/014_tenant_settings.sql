-- +goose Up

CREATE TABLE tenant_settings (
    tenant_id        UUID        PRIMARY KEY,
    default_currency TEXT        NOT NULL DEFAULT '',
    timezone         TEXT        NOT NULL DEFAULT '',
    date_format      TEXT        NOT NULL DEFAULT '',
    language         TEXT        NOT NULL DEFAULT '',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON tenant_settings
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down
DROP TABLE tenant_settings;
