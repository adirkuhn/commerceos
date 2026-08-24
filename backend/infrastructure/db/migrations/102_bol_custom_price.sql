-- +goose Up

-- Merchant's own sell price for a product's Bol offer, overriding
-- pms.SKU.PriceMinorUnits -- see domain/channels.BolProductOverride.
ALTER TABLE bol_product_overrides ADD COLUMN custom_price_minor_units BIGINT;

-- +goose Down
ALTER TABLE bol_product_overrides DROP COLUMN custom_price_minor_units;
