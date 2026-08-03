-- +goose Up

ALTER TABLE integration_connections ADD COLUMN default_currency TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE integration_connections DROP COLUMN default_currency;
