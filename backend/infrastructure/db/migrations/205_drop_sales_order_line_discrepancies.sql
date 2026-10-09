-- +goose Up
-- The "order differences" report (204) was dropped before use: Woo orders
-- are simply imported as they are (see application/channels.completeWooOrder),
-- and stock problems keep showing on Stock Issues as before.
DROP TABLE IF EXISTS sales_order_line_discrepancies;

-- +goose Down
-- Nothing to restore: 204's Down recreates nothing and the data had no use.
SELECT 1;
