-- +goose Up

ALTER TABLE products ADD COLUMN has_variants BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE products DROP COLUMN has_variants;
