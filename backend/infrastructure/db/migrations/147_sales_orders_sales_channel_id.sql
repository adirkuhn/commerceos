-- +goose Up
-- Records which internal channels.SalesChannel a SalesOrderSourceInternal
-- order was placed on -- needed once a tenant can run more than one
-- internal storefront (see domain/storefront's package doc comment).
-- NULL for every other source (only application/storefront's checkout
-- ever sets this).
ALTER TABLE sales_orders ADD COLUMN sales_channel_id UUID REFERENCES sales_channels(id) ON DELETE SET NULL;
CREATE INDEX sales_orders_sales_channel_id_idx ON sales_orders (sales_channel_id);

-- +goose Down
DROP INDEX IF EXISTS sales_orders_sales_channel_id_idx;
ALTER TABLE sales_orders DROP COLUMN sales_channel_id;
