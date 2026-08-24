-- +goose Up
-- Sales orders can now also arrive from Bol's own /orders endpoint
-- (ImportBolOrdersUseCase), same as WooCommerce's equivalent (migration 056).
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce', 'bol'));

-- +goose Down
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_source_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_source_check CHECK (source IN ('pos', 'manual', 'woocommerce'));
