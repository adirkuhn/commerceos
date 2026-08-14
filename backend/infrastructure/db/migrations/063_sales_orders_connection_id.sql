-- +goose Up
-- connection_id lets "delete all imported products" find and remove the
-- sales orders that belong to that specific connection first (so their
-- ON DELETE RESTRICT line items don't block the product delete) --
-- nullable since POS/manual orders have no connection at all.
ALTER TABLE sales_orders ADD COLUMN connection_id UUID REFERENCES integration_connections(id) ON DELETE SET NULL;
CREATE INDEX sales_orders_connection_id_idx ON sales_orders (connection_id);

-- Backfill existing Woo-sourced orders, but only for a tenant with exactly
-- one WooCommerce connection -- with more than one, which connection each
-- historical order came from is genuinely ambiguous (the column didn't
-- exist yet to record it), so those are deliberately left NULL rather than
-- guessed at. New orders are correctly stamped going forward regardless.
UPDATE sales_orders so
SET connection_id = c.id
FROM (
    -- MIN() has no uuid overload; cast through text and back. Only one row
    -- can exist per tenant_id here (HAVING COUNT(*) = 1), so which value
    -- MIN() would have picked never actually matters.
    SELECT tenant_id, MIN(id::text)::uuid AS id
    FROM integration_connections
    WHERE provider = 'woocommerce'
    GROUP BY tenant_id
    HAVING COUNT(*) = 1
) c
WHERE so.tenant_id = c.tenant_id
  AND so.source = 'woocommerce'
  AND so.connection_id IS NULL;

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN connection_id;
