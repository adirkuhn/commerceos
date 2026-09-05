-- +goose Up
-- Normalized E.164 phone number, distinct from Billing/Shipping.Phone
-- (free-text, WooCommerce-shaped) -- this is the field WhatsApp inbound
-- messages match against to resolve an existing customer by sender number.
-- Nullable/blank until set explicitly (manual edit, or the WhatsApp
-- new-customer capture flow) -- no backfill from billing/shipping phone,
-- since those aren't guaranteed E.164 and matching an un-normalized guess
-- risks a false positive.
ALTER TABLE customers ADD COLUMN phone_e164 TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX customers_tenant_phone_e164_idx ON customers (tenant_id, phone_e164) WHERE phone_e164 != '';

-- +goose Down
DROP INDEX IF EXISTS customers_tenant_phone_e164_idx;
ALTER TABLE customers DROP COLUMN phone_e164;
