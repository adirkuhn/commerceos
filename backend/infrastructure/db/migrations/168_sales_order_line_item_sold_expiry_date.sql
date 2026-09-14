-- +goose Up

-- Which expiry date WooCommerce reported as sold for this line, snapshotted
-- at import time from the commerceos-woo plugin's own order-item meta
-- (commerceos_expiry_date) -- see application/channels.ImportWooOrdersUseCase.
-- Null for every non-Woo line (POS, manual, other channels), and for a Woo
-- line whose product had no expiry_date meta set at checkout time. This is
-- what Woo itself believed was sold, not necessarily the same batch
-- CommerceOS's own FEFO allocation actually decremented (see
-- sales_order_line_item_stock_movements.batch_id for that) -- Woo only
-- ever knows one current lot/expiry value per product, never a real
-- multi-batch split.
ALTER TABLE sales_order_line_items ADD COLUMN sold_expiry_date DATE;

-- +goose Down
ALTER TABLE sales_order_line_items DROP COLUMN sold_expiry_date;
