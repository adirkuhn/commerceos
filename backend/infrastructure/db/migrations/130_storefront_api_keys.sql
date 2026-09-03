-- +goose Up
-- Multiple named, independently revocable API keys per tenant -- a
-- storefront (e.g. the WooCommerce plugin) presents one of these as a
-- bearer token to call CommerceOS' authenticated public-storefront API.
-- Supersedes the single anonymous rotate-only key on tenant_settings (see
-- migration 129): that shape doesn't let a merchant tell which storefront
-- is which, or revoke one without breaking the others.
CREATE TABLE storefront_api_keys (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    key_hash    TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX storefront_api_keys_key_hash_idx ON storefront_api_keys (key_hash);
CREATE INDEX storefront_api_keys_tenant_id_idx ON storefront_api_keys (tenant_id);

ALTER TABLE storefront_api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE storefront_api_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON storefront_api_keys
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Superseded by the table above -- multiple named keys per tenant instead
-- of one anonymous rotate-only key (see migration 129). IF EXISTS on both:
-- 129 was only ever applied on one dev environment, never in production,
-- so a fresh database here never had this column/index to begin with.
DROP INDEX IF EXISTS tenant_settings_storefront_key_hash_idx;
ALTER TABLE tenant_settings DROP COLUMN IF EXISTS storefront_key_hash;

-- +goose Down
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS storefront_key_hash TEXT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS tenant_settings_storefront_key_hash_idx ON tenant_settings (storefront_key_hash)
    WHERE storefront_key_hash IS NOT NULL;
DROP TABLE IF EXISTS storefront_api_keys;
