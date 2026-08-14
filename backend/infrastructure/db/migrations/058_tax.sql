-- +goose Up
-- Tax classes (e.g. "Standard", "Reduced rate", "Zero rate", or a tenant's
-- own custom class) each carry their own per-country rate table -- mirrors
-- WooCommerce's tax class model. A SKU references a class; a NULL
-- tax_class_id means "use the tenant's default class" rather than
-- duplicating the default's id onto every SKU row.
CREATE TABLE tax_classes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    name        TEXT        NOT NULL,
    is_default  BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tax_classes_tenant_id_idx ON tax_classes (tenant_id);

-- rate_basis_points: 1 bp = 0.01%, e.g. 2100 = 21.00% -- integer, same
-- rationale as the *_minor_units money columns elsewhere (no float
-- rounding on a value that ends up multiplied into money).
CREATE TABLE tax_rates (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    tax_class_id        UUID        NOT NULL REFERENCES tax_classes(id) ON DELETE CASCADE,
    country             TEXT        NOT NULL,
    rate_basis_points   INT         NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tax_rates_rate_basis_points_check CHECK (rate_basis_points >= 0),
    CONSTRAINT tax_rates_tenant_class_country_unique UNIQUE (tenant_id, tax_class_id, country)
);

CREATE INDEX tax_rates_tenant_id_idx ON tax_rates (tenant_id);
CREATE INDEX tax_rates_tax_class_id_idx ON tax_rates (tax_class_id);

ALTER TABLE tax_classes ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_classes FORCE ROW LEVEL SECURITY;
ALTER TABLE tax_rates ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_rates FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON tax_classes
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON tax_rates
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE tenant_settings ADD COLUMN prices_include_tax BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE tenant_settings ADD COLUMN default_tax_country TEXT NOT NULL DEFAULT '';

ALTER TABLE skus ADD COLUMN tax_class_id UUID REFERENCES tax_classes(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE skus DROP COLUMN tax_class_id;
ALTER TABLE tenant_settings DROP COLUMN default_tax_country;
ALTER TABLE tenant_settings DROP COLUMN prices_include_tax;
DROP TABLE tax_rates;
DROP TABLE tax_classes;
