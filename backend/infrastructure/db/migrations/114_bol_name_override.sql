-- +goose Up
ALTER TABLE bol_product_overrides ADD COLUMN name_override TEXT;

-- +goose Down
ALTER TABLE bol_product_overrides DROP COLUMN name_override;
