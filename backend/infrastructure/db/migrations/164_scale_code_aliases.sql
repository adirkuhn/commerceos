-- +goose Up
-- Additional scale codes for a SKU, alongside its primary skus.scale_code --
-- a tenant with more than one physical butcher/deli scale independently
-- programs each one, so the same product can end up with a different
-- 5-digit code per scale. Same shape/purpose as barcode_aliases (031),
-- but for scale codes rather than full barcodes.
CREATE TABLE scale_code_aliases (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    sku_id     UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    code       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);

CREATE INDEX scale_code_aliases_tenant_id_idx ON scale_code_aliases (tenant_id);
CREATE INDEX scale_code_aliases_sku_id_idx ON scale_code_aliases (sku_id);

ALTER TABLE scale_code_aliases ENABLE ROW LEVEL SECURITY;
ALTER TABLE scale_code_aliases FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON scale_code_aliases
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE scale_code_aliases;
