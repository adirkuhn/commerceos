-- +goose Up
-- Sales orders can now also arrive from Woo's own /orders endpoint
-- (ImportWooOrdersUseCase), not just POS pushes or manual admin entry.
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce'));

-- +goose Down
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual'));
