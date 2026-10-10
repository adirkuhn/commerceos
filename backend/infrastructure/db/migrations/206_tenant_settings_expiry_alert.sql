-- +goose Up

-- Daily stock-close-to-expiry report: on/off plus the chosen "days before
-- expiry" windows, a JSON array drawn from 90/30/15/7 (validated in
-- domain/settings.NewExpiryAlert). See domain/settings.ExpiryAlert.
ALTER TABLE tenant_settings
    ADD COLUMN expiry_alert_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN expiry_alert_windows JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE tenant_settings
    DROP COLUMN expiry_alert_windows,
    DROP COLUMN expiry_alert_enabled;
