-- +goose Up

ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('receipt', 'sale', 'adjustment', 'transfer_out', 'transfer_in', 'damaged', 'lost'));

-- +goose Down

ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('receipt', 'sale', 'adjustment', 'transfer_out', 'transfer_in'));
