-- +goose Up

ALTER TABLE sales_order_line_items ADD COLUMN barcode TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sales_order_line_items DROP COLUMN barcode;
