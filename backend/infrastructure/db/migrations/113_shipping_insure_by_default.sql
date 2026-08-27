-- +goose Up

-- Replaces a fixed "insurance amount (cents)" config value with a plain
-- on/off toggle -- the insured amount is no longer configured anywhere,
-- it's computed at shipment-creation time from that specific SalesOrder's
-- own TotalMinorUnits (CreateShipmentUseCase), so it can never go stale
-- against what's actually being shipped. Backfills the new boolean from
-- whether an amount was previously configured, best-effort.
ALTER TABLE shipping_connection_settings
    ADD COLUMN insure_by_default BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE shipping_connection_settings
    SET insure_by_default = TRUE
    WHERE default_insurance_amount_minor_units IS NOT NULL AND default_insurance_amount_minor_units > 0;

ALTER TABLE shipping_connection_settings
    DROP COLUMN default_insurance_amount_minor_units;

ALTER TABLE shipping_carrier_overrides
    ADD COLUMN insure_by_default BOOLEAN;

UPDATE shipping_carrier_overrides
    SET insure_by_default = (insurance_amount_minor_units IS NOT NULL AND insurance_amount_minor_units > 0)
    WHERE insurance_amount_minor_units IS NOT NULL;

ALTER TABLE shipping_carrier_overrides
    DROP COLUMN insurance_amount_minor_units;

-- +goose Down
ALTER TABLE shipping_carrier_overrides
    ADD COLUMN insurance_amount_minor_units INTEGER;

ALTER TABLE shipping_carrier_overrides
    DROP COLUMN insure_by_default;

ALTER TABLE shipping_connection_settings
    ADD COLUMN default_insurance_amount_minor_units INTEGER;

ALTER TABLE shipping_connection_settings
    DROP COLUMN insure_by_default;
