-- +goose Up

-- Bol.com's required EU-operator identifier, sent on every offer a Bol
-- connection pushes. NULL for every non-Bol connection (they never read
-- it) -- unlike webhook_secret_encrypted's '' sentinel, there's no
-- "unconfigured but still a valid empty string" reading here, so this is
-- a genuinely nullable column, not not-null-default-''.
ALTER TABLE integration_connections ADD COLUMN bol_economic_operator_id TEXT;

-- Per-(product, Bol connection) fields that either can't be derived from
-- pms.Product/SKU (gpc_code -- Bol's own category taxonomy, unrelated to
-- pms.Category) or need to be overridable per channel (ean_override,
-- commission_override_basis_points) -- see
-- domain/channels.BolProductOverride's doc comment for the full reasoning.
-- Connection-scoped like external_category_links, not provider-scoped.
CREATE TABLE bol_product_overrides (
    id                               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                        UUID        NOT NULL,
    product_id                       UUID        NOT NULL,
    connection_id                    UUID        NOT NULL,
    ean_override                     TEXT,
    gpc_code                         TEXT,
    condition_category               TEXT        NOT NULL DEFAULT 'NEW',
    fulfilment_method                TEXT        NOT NULL DEFAULT 'FBR',
    commission_override_basis_points INTEGER,
    reference_override               TEXT,
    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT bol_product_overrides_condition_check CHECK (condition_category IN ('NEW', 'SECONDHAND', 'REFURBISHED')),
    CONSTRAINT bol_product_overrides_fulfilment_check CHECK (fulfilment_method IN ('FBR', 'FBB', 'SHIPPING_VIA_BOL', 'BOL_DELIVERY_PROMISE')),
    UNIQUE (product_id, connection_id)
);

CREATE INDEX bol_product_overrides_tenant_id_idx ON bol_product_overrides (tenant_id);

ALTER TABLE bol_product_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE bol_product_overrides FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON bol_product_overrides
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE bol_product_overrides;
ALTER TABLE integration_connections DROP COLUMN bol_economic_operator_id;
