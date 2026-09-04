-- +goose Up
ALTER TABLE tenant_settings ADD COLUMN recommendations_min_support INTEGER NOT NULL DEFAULT 2;

-- +goose Down
ALTER TABLE tenant_settings DROP COLUMN recommendations_min_support;
