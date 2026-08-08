-- +goose Up
-- A SKU can represent a sealed case/box of another SKU (e.g. a 36-unit case
-- of "Rapadurinha de Minas 25g"). Both columns are set together or not at
-- all (enforced in application/pms's CreateSKUUseCase/UpdateSKUUseCase, see
-- ErrCaseOfSKURequiresUnitsPerCase) -- purely a catalog relationship used to
-- split case-stock into unit-stock later (see
-- application/inventory.SplitCaseIntoUnitsUseCase).
ALTER TABLE skus ADD COLUMN case_of_sku_id UUID NULL REFERENCES skus(id) ON DELETE SET NULL;
ALTER TABLE skus ADD COLUMN units_per_case INTEGER NULL;

-- +goose Down
ALTER TABLE skus DROP COLUMN units_per_case;
ALTER TABLE skus DROP COLUMN case_of_sku_id;
