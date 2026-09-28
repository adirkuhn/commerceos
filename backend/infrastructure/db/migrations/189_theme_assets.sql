-- +goose Up
-- Tenant-wide theme media library. Documents refer to assets as
-- "asset:<id>"; deleting only sets deleted_at, since published theme
-- versions may still show the image. See domain/theme.Asset.
CREATE TABLE theme_assets (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    url          TEXT        NOT NULL,
    filename     TEXT        NOT NULL DEFAULT '',
    content_type TEXT        NOT NULL,
    bytes        BIGINT      NOT NULL,
    width        INT         NOT NULL DEFAULT 0,
    height       INT         NOT NULL DEFAULT 0,
    alt          TEXT        NOT NULL DEFAULT '',
    created_by   TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX theme_assets_tenant_created_idx ON theme_assets (tenant_id, created_at DESC) WHERE deleted_at IS NULL;

ALTER TABLE theme_assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_assets FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON theme_assets
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE theme_assets;
