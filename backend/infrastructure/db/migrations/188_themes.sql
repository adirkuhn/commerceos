-- +goose Up
-- Versioned storefront themes. See domain/theme and
-- docs/storefront-architecture.md.
CREATE TABLE themes (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    sales_channel_id     UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    name                 TEXT        NOT NULL,
    preset_key           TEXT        NOT NULL,
    published_version_id UUID,
    is_live              BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT themes_live_needs_published_version CHECK (NOT is_live OR published_version_id IS NOT NULL)
);
CREATE INDEX themes_tenant_id_idx ON themes (tenant_id);
CREATE INDEX themes_sales_channel_id_idx ON themes (sales_channel_id);
-- At most one live theme per channel.
CREATE UNIQUE INDEX themes_one_live_per_channel ON themes (sales_channel_id) WHERE is_live;

CREATE TABLE theme_versions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    theme_id   UUID        NOT NULL REFERENCES themes(id) ON DELETE CASCADE,
    number     INT         NOT NULL,
    state      TEXT        NOT NULL CHECK (state IN ('draft', 'sealed')),
    revision   INT         NOT NULL DEFAULT 1,
    settings   JSONB       NOT NULL,
    note       TEXT        NOT NULL DEFAULT '',
    created_by TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sealed_at  TIMESTAMPTZ,
    CONSTRAINT theme_versions_theme_number_unique UNIQUE (theme_id, number),
    CONSTRAINT theme_versions_sealed_at CHECK ((state = 'sealed') = (sealed_at IS NOT NULL))
);
CREATE INDEX theme_versions_tenant_id_idx ON theme_versions (tenant_id);
-- At most one draft per theme.
CREATE UNIQUE INDEX theme_versions_one_draft ON theme_versions (theme_id) WHERE state = 'draft';

ALTER TABLE themes ADD CONSTRAINT themes_published_version_fk
    FOREIGN KEY (published_version_id) REFERENCES theme_versions(id) ON DELETE SET NULL;

CREATE TABLE theme_documents (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    version_id UUID        NOT NULL REFERENCES theme_versions(id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL CHECK (kind IN ('template', 'page', 'section_group')),
    key        TEXT        NOT NULL,
    title      TEXT        NOT NULL DEFAULT '',
    seo        JSONB       NOT NULL DEFAULT '{}',
    tree       JSONB       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT theme_documents_version_kind_key_unique UNIQUE (version_id, kind, key)
);
CREATE INDEX theme_documents_tenant_id_idx ON theme_documents (tenant_id);

CREATE TABLE theme_sessions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    theme_id   UUID        NOT NULL REFERENCES themes(id) ON DELETE CASCADE,
    user_id    TEXT        NOT NULL,
    role       TEXT        NOT NULL,
    purpose    TEXT        NOT NULL CHECK (purpose IN ('edit', 'preview')),
    token_hash TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT theme_sessions_token_hash_unique UNIQUE (token_hash)
);
CREATE INDEX theme_sessions_tenant_id_idx ON theme_sessions (tenant_id);

-- Published versions are immutable. The domain enforces it too; this
-- makes it hold even for a buggy or hand-written query. DELETE stays
-- allowed so deleting a theme can cascade.
-- +goose StatementBegin
CREATE FUNCTION theme_versions_block_sealed_update() RETURNS trigger AS $$
BEGIN
    IF OLD.state = 'sealed' THEN
        RAISE EXCEPTION 'theme version % is published and cannot be changed', OLD.id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER theme_versions_block_sealed_update
    BEFORE UPDATE ON theme_versions FOR EACH ROW EXECUTE FUNCTION theme_versions_block_sealed_update();

-- +goose StatementBegin
CREATE FUNCTION theme_documents_block_sealed_write() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM theme_versions WHERE id = NEW.version_id AND state = 'sealed') THEN
        RAISE EXCEPTION 'theme version % is published and cannot be changed', NEW.version_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER theme_documents_block_sealed_write
    BEFORE INSERT OR UPDATE ON theme_documents FOR EACH ROW EXECUTE FUNCTION theme_documents_block_sealed_write();

ALTER TABLE themes ENABLE ROW LEVEL SECURITY;
ALTER TABLE themes FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON themes
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE theme_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_versions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON theme_versions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE theme_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON theme_documents
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE theme_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON theme_sessions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE theme_sessions;
DROP TABLE theme_documents;
ALTER TABLE themes DROP CONSTRAINT themes_published_version_fk;
DROP TABLE theme_versions;
DROP TABLE themes;
DROP FUNCTION theme_documents_block_sealed_write();
DROP FUNCTION theme_versions_block_sealed_update();
