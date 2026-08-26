-- +goose Up

-- Per-carrier (and connection-default) insurance amount, plus config
-- needed for the MyParcel Orders/Fulfilment API export step
-- (myparcel_account_id/shop_id -- the tenant's own MyParcel merchant
-- identifiers, not secrets) and auto-print (auto_print_enabled -- see
-- ShippingConnectionSettings.AutoPrintEnabled's doc for what "auto" means
-- here: auto-opening the browser print dialog, not silent printer
-- targeting).
ALTER TABLE shipping_connection_settings
    ADD COLUMN default_insurance_amount_minor_units INTEGER,
    ADD COLUMN myparcel_account_id INTEGER,
    ADD COLUMN myparcel_shop_id INTEGER,
    ADD COLUMN auto_print_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE shipping_carrier_overrides
    ADD COLUMN insurance_amount_minor_units INTEGER;

-- +goose Down
ALTER TABLE shipping_carrier_overrides
    DROP COLUMN insurance_amount_minor_units;

ALTER TABLE shipping_connection_settings
    DROP COLUMN default_insurance_amount_minor_units,
    DROP COLUMN myparcel_account_id,
    DROP COLUMN myparcel_shop_id,
    DROP COLUMN auto_print_enabled;
