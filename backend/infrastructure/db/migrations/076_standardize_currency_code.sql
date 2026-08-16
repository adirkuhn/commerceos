-- +goose Up

-- skus.currency_code was CHAR(3)/bpchar; every other currency_code column
-- (sales_orders, purchase_orders, tenant_settings.default_currency,
-- sync_schedules.default_currency, import_jobs.default_currency) is plain
-- TEXT. No Go code anywhere handles bpchar's trailing-space padding
-- specially, so nothing was actually relying on the fixed-width type --
-- standardizing on TEXT everywhere, with an explicit CHECK replacing what
-- the fixed width used to (loosely) enforce.
ALTER TABLE skus ALTER COLUMN currency_code TYPE TEXT;

ALTER TABLE skus ADD CONSTRAINT skus_currency_code_format CHECK (currency_code ~ '^[A-Z]{3}$');
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_currency_code_format CHECK (currency_code ~ '^[A-Z]{3}$');
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_currency_code_format CHECK (currency_code ~ '^[A-Z]{3}$');
ALTER TABLE tenant_settings ADD CONSTRAINT tenant_settings_default_currency_format CHECK (default_currency ~ '^[A-Z]{3}$');
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_default_currency_format CHECK (default_currency IS NULL OR default_currency ~ '^[A-Z]{3}$');
ALTER TABLE import_jobs ADD CONSTRAINT import_jobs_default_currency_format CHECK (default_currency ~ '^[A-Z]{3}$');

-- +goose Down

ALTER TABLE import_jobs DROP CONSTRAINT import_jobs_default_currency_format;
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_default_currency_format;
ALTER TABLE tenant_settings DROP CONSTRAINT tenant_settings_default_currency_format;
ALTER TABLE purchase_orders DROP CONSTRAINT purchase_orders_currency_code_format;
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_currency_code_format;
ALTER TABLE skus DROP CONSTRAINT skus_currency_code_format;
ALTER TABLE skus ALTER COLUMN currency_code TYPE CHAR(3);
