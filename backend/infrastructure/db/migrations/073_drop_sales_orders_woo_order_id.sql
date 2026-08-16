-- +goose Up

-- woo_order_id duplicated external_order_id (same fact, different type --
-- external_order_id is strconv.Itoa() of the same int for a Woo order)
-- with nothing keeping them in sync. Write-only in application code: never
-- read back, never exposed via any API response. Dropped in favor of the
-- one column that's already the source of truth for every source.
ALTER TABLE sales_orders DROP COLUMN woo_order_id;

-- +goose Down
ALTER TABLE sales_orders ADD COLUMN woo_order_id BIGINT;
