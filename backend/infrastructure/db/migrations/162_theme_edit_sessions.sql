-- +goose Up
-- A short-lived, admin-minted bearer credential letting storefront-
-- frontend's embedded Puck editor read/write one channel's theme pages
-- without ever holding the merchant's own JWT. See
-- domain/theme.EditSession's doc comment for why this is a distinct
-- credential from a storefront api_keys row (that one only ever grants
-- public/anonymous-shopper privilege).
CREATE TABLE theme_edit_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    token            TEXT        NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT theme_edit_sessions_token_unique UNIQUE (token)
);

CREATE INDEX theme_edit_sessions_tenant_id_idx ON theme_edit_sessions (tenant_id);

ALTER TABLE theme_edit_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_edit_sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON theme_edit_sessions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE theme_edit_sessions;
