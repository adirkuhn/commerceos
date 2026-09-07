-- +goose Up

-- Tracks how much of a line item has been refunded (partial-by-quantity --
-- see RefundLineItemUseCase) -- accumulates across possibly more than one
-- refund scan against the same line.
ALTER TABLE sales_order_line_items ADD COLUMN refunded_quantity BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_order_line_items ADD COLUMN refunded_amount_minor_units BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sales_order_line_items DROP COLUMN refunded_amount_minor_units;
ALTER TABLE sales_order_line_items DROP COLUMN refunded_quantity;
