-- +goose Up
-- The shop's admin appearance (domain/settings.Appearance): font, accent
-- and sidebar colours, corners, text size, density -- applied to every
-- user's screen in the new design. One row per tenant; no row means the
-- defaults (settings.DefaultAppearance). Values are validated in the
-- domain, not here, so a new option needs no migration.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE tenant_appearance (
    tenant_id     UUID        PRIMARY KEY NOT NULL,
    font          TEXT        NOT NULL,
    accent_color  TEXT        NOT NULL,
    sidebar_color TEXT        NOT NULL,
    corners       TEXT        NOT NULL,
    text_size     TEXT        NOT NULL,
    density       TEXT        NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

ALTER TABLE tenant_appearance ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_appearance FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON tenant_appearance
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS tenant_appearance;
