-- +goose Up
ALTER TABLE tenant_settings ADD COLUMN recommendations_lookback_days INTEGER NOT NULL DEFAULT 180;

-- +goose Down
ALTER TABLE tenant_settings DROP COLUMN recommendations_lookback_days;
