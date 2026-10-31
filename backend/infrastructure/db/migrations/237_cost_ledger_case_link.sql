-- +goose Up
-- A unit SKU sold out of a sealed case (skus.case_of_sku_id) takes its
-- cost from the case: case cost / units_per_case (see
-- application/pms.SyncUnitCostFromCaseUseCase). Each such change is kept
-- in the cost ledger as its own event.
ALTER TABLE sku_cost_price_ledger DROP CONSTRAINT sku_cost_price_ledger_source_type_check;
ALTER TABLE sku_cost_price_ledger ADD CONSTRAINT sku_cost_price_ledger_source_type_check
    CHECK (source_type IN ('opening', 'receipt', 'sale', 'return', 'damaged', 'lost', 'adjustment', 'correction', 'merge', 'case_link'));

-- +goose Down
DELETE FROM sku_cost_price_ledger WHERE source_type = 'case_link';
ALTER TABLE sku_cost_price_ledger DROP CONSTRAINT sku_cost_price_ledger_source_type_check;
ALTER TABLE sku_cost_price_ledger ADD CONSTRAINT sku_cost_price_ledger_source_type_check
    CHECK (source_type IN ('opening', 'receipt', 'sale', 'return', 'damaged', 'lost', 'adjustment', 'correction', 'merge'));
