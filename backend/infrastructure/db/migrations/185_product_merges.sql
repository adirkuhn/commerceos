-- +goose Up
-- Merging a duplicate product into an existing one (see
-- application/pms.MergeProductsUseCase): every SKU reference moves to the
-- matched existing SKU and the duplicate is deleted. product_merges is the
-- audit trail of what was merged into what (the duplicate row itself is
-- gone afterwards, so its name and SKU codes are kept here).
CREATE TABLE product_merges (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    source_product_id   UUID        NOT NULL,
    source_product_name TEXT        NOT NULL,
    target_product_id   UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    -- [{source_sku_id, source_sku_code, source_barcode, target_sku_id}]
    sku_mapping         JSONB       NOT NULL,
    merged_by           TEXT        NOT NULL,
    merged_at           TIMESTAMPTZ NOT NULL
);

CREATE INDEX product_merges_tenant_target_idx ON product_merges (tenant_id, target_product_id);

ALTER TABLE product_merges ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_merges FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON product_merges
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- A merge blends the two SKUs' moving-average costs into one: logged as
-- its own ledger event.
ALTER TABLE sku_cost_price_ledger DROP CONSTRAINT sku_cost_price_ledger_source_type_check;
ALTER TABLE sku_cost_price_ledger ADD CONSTRAINT sku_cost_price_ledger_source_type_check
    CHECK (source_type IN ('opening', 'receipt', 'sale', 'return', 'damaged', 'lost', 'adjustment', 'correction', 'merge'));

-- +goose Down
-- Rolling back drops merge ledger rows: the old constraint can't hold them.
DELETE FROM sku_cost_price_ledger WHERE source_type = 'merge';
ALTER TABLE sku_cost_price_ledger DROP CONSTRAINT sku_cost_price_ledger_source_type_check;
ALTER TABLE sku_cost_price_ledger ADD CONSTRAINT sku_cost_price_ledger_source_type_check
    CHECK (source_type IN ('opening', 'receipt', 'sale', 'return', 'damaged', 'lost', 'adjustment', 'correction'));
DROP TABLE product_merges;
