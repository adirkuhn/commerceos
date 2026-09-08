-- +goose Up
-- SalesChannel.DefaultWarehouseID -- mirrors integrations.Connection's own
-- DefaultWarehouseID (used for WooCommerce/Bol import), but for channel
-- types with no external connection to hang it off (Internal, WhatsApp):
-- both need a real warehouse to attribute a placed order's stock
-- decrement to (sales_orders.warehouse_id is NOT NULL), and neither had
-- any way to configure one -- found live: every internal-storefront or
-- WhatsApp checkout failed at the DB with "invalid input syntax for type
-- uuid: ''" the moment a real order was placed.
ALTER TABLE sales_channels ADD COLUMN default_warehouse_id UUID REFERENCES warehouses(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE sales_channels DROP COLUMN default_warehouse_id;
