-- +goose Up

-- payment_provider/payment_external_id capture the provider (stripe/revolut)
-- and its own charge/session reference at webhook-confirm time (see
-- application/payments.WebhookDispatchHandler) -- previously nothing stored
-- this, so a refund had nothing to call the provider's API against.
-- needs_manual_refund_review flags an order whose refund couldn't be issued
-- through the provider's API (no provider info -- e.g. an imported order --
-- or the API call itself failed) so a real PickScanActionRefund still
-- records the item as refunded, just not automatically.
ALTER TABLE sales_orders ADD COLUMN payment_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN payment_external_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN needs_manual_refund_review BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check
    CHECK (status IN ('pending', 'processing', 'on-hold', 'shipped', 'completed', 'cancelled', 'refunded', 'failed', 'partially_refunded'));

-- +goose Down
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check
    CHECK (status IN ('pending', 'processing', 'on-hold', 'shipped', 'completed', 'cancelled', 'refunded', 'failed'));

ALTER TABLE sales_orders DROP COLUMN needs_manual_refund_review;
ALTER TABLE sales_orders DROP COLUMN payment_external_id;
ALTER TABLE sales_orders DROP COLUMN payment_provider;
