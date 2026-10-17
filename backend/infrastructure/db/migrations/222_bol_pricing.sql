-- +goose Up

-- What one parcel costs a Bol connection's merchant. Automatic Bol prices
-- add it on top of the regular price (shared across the units of a bulk
-- tier) -- see domain/channels.ResolveBolPrices. 0 = no shipping cost.
ALTER TABLE integration_connections
    ADD COLUMN bol_shipping_cost_minor_units BIGINT NOT NULL DEFAULT 0
        CONSTRAINT integration_connections_bol_shipping_cost_check CHECK (bol_shipping_cost_minor_units >= 0);

-- Per-product Bol bulk prices: [{"Quantity": 2, "UnitPriceMinorUnits": null}, ...],
-- a null price is automatic. Only sent to Bol while bundle_pricing_enabled.
ALTER TABLE bol_product_overrides
    ADD COLUMN bundle_pricing_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN bundle_tiers JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE bol_product_overrides DROP COLUMN bundle_tiers, DROP COLUMN bundle_pricing_enabled;
ALTER TABLE integration_connections DROP COLUMN bol_shipping_cost_minor_units;
