-- +goose Up
ALTER TABLE skus ADD COLUMN sale_price_minor_units BIGINT NULL;
ALTER TABLE skus ADD CONSTRAINT skus_sale_price_not_over_price
    CHECK (sale_price_minor_units IS NULL OR sale_price_minor_units <= price_minor_units);

-- +goose Down
ALTER TABLE skus DROP CONSTRAINT skus_sale_price_not_over_price;
ALTER TABLE skus DROP COLUMN sale_price_minor_units;
