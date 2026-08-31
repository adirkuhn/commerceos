-- +goose Up

-- Inbound order_status_change webhooks carry MyParcel's own order UUID and
-- need the reverse lookup back to (connection_id, sales_order_id) --
-- FindByMyParcelUUID. UNIQUE, not just an index: MyParcel's own uuid is
-- generated per export call and this table's own UNIQUE(connection_id,
-- sales_order_id) already guarantees at most one export per order, so two
-- rows sharing a myparcel_uuid would mean MyParcel handed back a duplicate
-- id -- something this lookup should fail loudly on, not silently pick one.
CREATE UNIQUE INDEX myparcel_fulfilment_orders_myparcel_uuid_idx ON myparcel_fulfilment_orders (myparcel_uuid);

-- +goose Down
DROP INDEX myparcel_fulfilment_orders_myparcel_uuid_idx;
