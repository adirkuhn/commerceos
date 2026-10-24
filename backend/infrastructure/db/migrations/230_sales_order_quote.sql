-- +goose Up
-- The quote a sales order was created from (Quotes -> Create sales order).
-- One quote can become several orders; deleting the quote keeps the orders.
ALTER TABLE sales_orders ADD COLUMN quote_id UUID REFERENCES quotes(id) ON DELETE SET NULL;
CREATE INDEX sales_orders_quote_idx ON sales_orders (quote_id) WHERE quote_id IS NOT NULL;

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN IF EXISTS quote_id;
