-- +goose Up
-- COM-11: a SKU's cost price becomes the landed cost of its last goods
-- receipt (replacement cost, for pricing and margins). The moving weighted
-- average -- what the stock on hand actually cost, used for COGS and stock
-- value -- moves to its own column. Until now cost_price_minor_units held
-- that average, so it seeds the new column. Additive: old code ignores it.
ALTER TABLE skus ADD COLUMN inventory_cost_minor_units BIGINT;
UPDATE skus SET inventory_cost_minor_units = cost_price_minor_units WHERE cost_price_minor_units IS NOT NULL;

-- +goose Down
ALTER TABLE skus DROP COLUMN inventory_cost_minor_units;
