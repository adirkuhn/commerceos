-- +goose Up
-- Passwordless storefront customer login (domain/storefront.MagicLinkToken
-- / CustomerSession) -- a customer clicks an emailed one-time link and
-- gets a long-lived session token, same identity model as Cart/Wishlist
-- (opaque token, no password anywhere).
CREATE TABLE storefront_magic_link_tokens (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    customer_id      UUID        NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    token            TEXT        NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    used_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX storefront_magic_link_tokens_token_idx ON storefront_magic_link_tokens (token);
CREATE INDEX storefront_magic_link_tokens_tenant_id_idx ON storefront_magic_link_tokens (tenant_id);

ALTER TABLE storefront_magic_link_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE storefront_magic_link_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON storefront_magic_link_tokens
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE TABLE storefront_customer_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    customer_id      UUID        NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    token            TEXT        NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX storefront_customer_sessions_token_idx ON storefront_customer_sessions (token);
CREATE INDEX storefront_customer_sessions_tenant_id_idx ON storefront_customer_sessions (tenant_id);

ALTER TABLE storefront_customer_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE storefront_customer_sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON storefront_customer_sessions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE storefront_customer_sessions;
DROP TABLE storefront_magic_link_tokens;
