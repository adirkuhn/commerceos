-- +goose Up
-- Per-tenant key a storefront (e.g. the WooCommerce plugin) presents as a
-- bearer token to call the authenticated public-storefront API (cart
-- recommendations first). Hash-only, like a password: once issued we only
-- ever need to compare it, never recover the plaintext. Lives on
-- tenant_settings (tenant-wide), not on a specific integration connection --
-- a tenant's storefront calls aren't scoped to any one connection, and a
-- tenant can have several (see ConnectionRepository.ListByTenant).
ALTER TABLE tenant_settings ADD COLUMN storefront_key_hash TEXT NULL;

CREATE UNIQUE INDEX tenant_settings_storefront_key_hash_idx ON tenant_settings (storefront_key_hash)
    WHERE storefront_key_hash IS NOT NULL;

-- +goose Down
DROP INDEX tenant_settings_storefront_key_hash_idx;
ALTER TABLE tenant_settings DROP COLUMN storefront_key_hash;
