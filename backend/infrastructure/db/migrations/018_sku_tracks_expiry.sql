-- +goose Up

ALTER TABLE skus ADD COLUMN tracks_expiry BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE skus DROP COLUMN tracks_expiry;
