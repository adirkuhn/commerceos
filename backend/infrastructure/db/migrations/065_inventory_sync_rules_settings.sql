-- +goose Up

ALTER TABLE tenant_settings
    ADD COLUMN inventory_sync_rules_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN batch_expiry_threshold_days  INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tenant_settings
    DROP COLUMN inventory_sync_rules_enabled,
    DROP COLUMN batch_expiry_threshold_days;
