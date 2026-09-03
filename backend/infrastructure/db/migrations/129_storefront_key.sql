-- +goose Up
-- Per-connection key a storefront (e.g. the WooCommerce plugin) presents as
-- a bearer token to call the authenticated public-storefront API (cart
-- recommendations first). Hash-only, like a password: once issued we only
-- ever need to compare it, never recover the plaintext, unlike
-- consumer_secret_encrypted/webhook_secret_encrypted which must be
-- reversible. Nullable -- most connections never generate one.
ALTER TABLE integration_connections ADD COLUMN storefront_key_hash TEXT NULL;

CREATE UNIQUE INDEX integration_connections_storefront_key_hash_idx ON integration_connections (storefront_key_hash)
    WHERE storefront_key_hash IS NOT NULL;

-- +goose Down
DROP INDEX integration_connections_storefront_key_hash_idx;
ALTER TABLE integration_connections DROP COLUMN storefront_key_hash;
