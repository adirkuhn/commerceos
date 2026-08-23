-- +goose Up

-- tenant_settings.default_currency uses '' as its "unset" sentinel (see
-- domain/settings/tenant_settings.go), same as sync_schedules.default_currency
-- uses NULL for the same purpose. 076_standardize_currency_code.sql gave
-- sync_schedules an `IS NULL OR ...` escape hatch for its sentinel but missed
-- giving tenant_settings the equivalent `= '' OR ...` escape hatch, so any
-- write that leaves currency unset (e.g. a partial settings PATCH) violates
-- the CHECK.
ALTER TABLE tenant_settings DROP CONSTRAINT tenant_settings_default_currency_format;
ALTER TABLE tenant_settings ADD CONSTRAINT tenant_settings_default_currency_format
    CHECK (default_currency = '' OR default_currency ~ '^[A-Z]{3}$');

-- +goose Down

ALTER TABLE tenant_settings DROP CONSTRAINT tenant_settings_default_currency_format;
ALTER TABLE tenant_settings ADD CONSTRAINT tenant_settings_default_currency_format
    CHECK (default_currency ~ '^[A-Z]{3}$');
