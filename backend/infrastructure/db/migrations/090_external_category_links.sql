-- +goose Up
-- Persisted, editable mapping from one Woo store connection's own category
-- id to one of this tenant's pms.Category rows. Connection-scoped (not
-- provider-scoped like external_tax_class_links) -- Woo category ids are
-- per-store, so a tenant with two Woo connections needs independent
-- mappings, not one shared by provider alone. A row is created either
-- automatically (product push finds a live Woo category with a matching
-- slug and no link yet) or manually (Settings mapping UI).
CREATE TABLE external_category_links (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    connection_id UUID        NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    provider      TEXT        NOT NULL,
    external_id   TEXT        NOT NULL,
    category_id   UUID        NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT external_category_links_connection_external_unique UNIQUE (connection_id, external_id),
    CONSTRAINT external_category_links_connection_category_unique UNIQUE (connection_id, category_id)
);

CREATE INDEX external_category_links_tenant_id_idx ON external_category_links (tenant_id);
CREATE INDEX external_category_links_category_id_idx ON external_category_links (category_id);

ALTER TABLE external_category_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE external_category_links FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON external_category_links
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE external_category_links;
