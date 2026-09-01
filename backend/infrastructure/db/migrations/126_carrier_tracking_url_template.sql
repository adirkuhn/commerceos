-- +goose Up
-- Staff-configured per-carrier tracking-page URL template (see
-- domain/shipping.ShippingCarrierOverride.TrackingURLTemplate's doc) --
-- preferred over MyParcel's own link_consumer_portal when set, since that
-- isn't always present or correct for every carrier.
ALTER TABLE shipping_carrier_overrides ADD COLUMN tracking_url_template TEXT;

-- +goose Down
ALTER TABLE shipping_carrier_overrides DROP COLUMN tracking_url_template;
