-- +goose Up
ALTER TABLE skus ADD COLUMN sale_end_date DATE NULL;
ALTER TABLE skus ADD CONSTRAINT skus_sale_end_date_requires_sale_price
    CHECK (sale_end_date IS NULL OR sale_price_minor_units IS NOT NULL);

-- +goose Down
ALTER TABLE skus DROP CONSTRAINT skus_sale_end_date_requires_sale_price;
ALTER TABLE skus DROP COLUMN sale_end_date;
