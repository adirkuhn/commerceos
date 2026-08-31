-- +goose Up
-- MyParcel's tracktrace response includes link_consumer_portal -- the
-- carrier-specific tracking page (confirmed against
-- developer.myparcel.nl/api-reference/06.shipments.html) meant to be shared
-- with the end customer. Persisted alongside status/status_main so it's
-- available later (e.g. the Woo shipment-info order note, see
-- application/channels.PushSalesOrderStatusToChannelsUseCase) without a
-- live MyParcel call.
ALTER TABLE shipments ADD COLUMN tracking_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE shipments DROP COLUMN tracking_url;
