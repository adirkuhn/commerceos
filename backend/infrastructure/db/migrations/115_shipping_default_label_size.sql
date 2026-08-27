-- +goose Up
ALTER TABLE shipping_connection_settings ADD COLUMN default_label_size TEXT NOT NULL DEFAULT 'A4';

-- +goose Down
ALTER TABLE shipping_connection_settings DROP COLUMN default_label_size;
