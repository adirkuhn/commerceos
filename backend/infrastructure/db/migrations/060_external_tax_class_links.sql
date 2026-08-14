-- +goose Up
-- Persisted, editable mapping from an external channel's own tax-class
-- identifier (e.g. WooCommerce's tax_class slug, "" = its Standard rate)
-- to one of this tenant's domain/tax.TaxClass rows. Import auto-creates a
-- row the first time it sees a new external slug (see
-- ImportWooProductsUseCase); staff can repoint it in Settings -> Tax
-- afterward without editing the class itself.
CREATE TABLE external_tax_class_links (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    provider      TEXT        NOT NULL,
    external_slug TEXT        NOT NULL,
    tax_class_id  UUID        NOT NULL REFERENCES tax_classes(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT external_tax_class_links_tenant_provider_slug_unique UNIQUE (tenant_id, provider, external_slug)
);

CREATE INDEX external_tax_class_links_tenant_id_idx ON external_tax_class_links (tenant_id);
CREATE INDEX external_tax_class_links_tax_class_id_idx ON external_tax_class_links (tax_class_id);

ALTER TABLE external_tax_class_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE external_tax_class_links FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON external_tax_class_links
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE external_tax_class_links;
