-- +goose Up

ALTER TABLE skus ADD COLUMN is_virtual BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE skus DROP COLUMN is_virtual;
