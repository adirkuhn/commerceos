-- +goose Up
-- split_out/split_in: SplitCaseIntoUnitsUseCase's atomic case->units stock
-- conversion (see pms.SKU.CaseOfSKUID) -- always written as a pair, one
-- movement per SKU (the case SKU loses stock, its unit SKU gains
-- UnitsPerCase times as much).
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('receipt', 'sale', 'adjustment', 'transfer_out', 'transfer_in', 'damaged', 'lost', 'split_out', 'split_in'));

-- +goose Down
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('receipt', 'sale', 'adjustment', 'transfer_out', 'transfer_in', 'damaged', 'lost'));
