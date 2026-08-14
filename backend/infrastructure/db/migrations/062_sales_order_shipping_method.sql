-- +goose Up
ALTER TABLE sales_orders ADD COLUMN shipping_method TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_method_title TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN shipping_method_title;
ALTER TABLE sales_orders DROP COLUMN shipping_method;
