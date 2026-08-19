-- +goose Up

-- Woo order import (ImportWooOrdersUseCase) and status-driven decrements
-- (UpdateSalesOrderUseCase) used to hard-fail the whole order when a line
-- item's stock decrement hit ErrInsufficientStock, or silently omit an
-- unmatched-SKU line item from ever decrementing -- both left the order
-- stuck with no way for staff to resolve it short of a manual DB fix.
-- stock_status tracks, per line item, whether its stock-reducing
-- obligation (if the order's status owes one) has actually been carried
-- out: 'pending' means it hasn't (either the decrement failed for
-- insufficient stock, or the line has no sku_id yet to decrement against).
-- needs_stock_review/needs_sku_review are the order-level rollup staff
-- filter on (see the new /sales-orders?needs_stock_review=/needs_sku_review=
-- list filters and the retry-stock/link-sku endpoints that clear them).
ALTER TABLE sales_order_line_items ADD COLUMN stock_status TEXT NOT NULL DEFAULT 'decremented'
    CHECK (stock_status IN ('decremented', 'pending'));

ALTER TABLE sales_orders ADD COLUMN needs_stock_review BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE sales_orders ADD COLUMN needs_sku_review BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN needs_sku_review;
ALTER TABLE sales_orders DROP COLUMN needs_stock_review;
ALTER TABLE sales_order_line_items DROP COLUMN stock_status;
