-- +goose Up
-- Low stock: a product counts as low when its total stock is above 0 and at
-- or below its threshold -- the product's own low_stock_threshold when set,
-- else the tenant's (Settings -> Inventory). DEFAULT 10 keeps the rule the
-- inventory screen used before this was configurable. Thresholds are in the
-- stock's own unit: pieces, or grams for weighed (kilo) SKUs.
ALTER TABLE tenant_settings
    ADD COLUMN low_stock_threshold INTEGER NOT NULL DEFAULT 10 CHECK (low_stock_threshold >= 0);

ALTER TABLE products
    ADD COLUMN low_stock_threshold INTEGER NULL CHECK (low_stock_threshold >= 0);

-- +goose Down
ALTER TABLE products DROP COLUMN low_stock_threshold;
ALTER TABLE tenant_settings DROP COLUMN low_stock_threshold;
