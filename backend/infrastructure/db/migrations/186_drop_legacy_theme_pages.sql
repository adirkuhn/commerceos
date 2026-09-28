-- +goose Up
-- The Puck/freeform theme builder (theme_pages, theme_edit_sessions) is
-- replaced by the versioned theme system (docs/storefront-architecture.md).
-- Its admin UI was already removed on 2026-09-11; nothing reads these
-- tables any more.
DROP TABLE IF EXISTS theme_edit_sessions;
DROP TABLE IF EXISTS theme_pages;

-- +goose Down
CREATE TABLE theme_pages (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    slug             TEXT        NOT NULL,
    canvas_width     INT         NOT NULL DEFAULT 1440,
    blocks           JSONB       NOT NULL DEFAULT '[]',
    custom_css       TEXT        NOT NULL DEFAULT '',
    editor_mode      TEXT        NOT NULL DEFAULT 'freeform',
    puck_data        JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT theme_pages_channel_slug_unique UNIQUE (sales_channel_id, slug),
    CONSTRAINT theme_pages_editor_mode_check CHECK (editor_mode IN ('freeform', 'puck'))
);
CREATE INDEX theme_pages_tenant_id_idx ON theme_pages (tenant_id);
ALTER TABLE theme_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_pages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON theme_pages
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE TABLE theme_edit_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    token            TEXT        NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT theme_edit_sessions_token_unique UNIQUE (token)
);
CREATE INDEX theme_edit_sessions_tenant_id_idx ON theme_edit_sessions (tenant_id);
ALTER TABLE theme_edit_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_edit_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON theme_edit_sessions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);
