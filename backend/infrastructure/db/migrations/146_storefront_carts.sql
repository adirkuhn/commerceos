-- +goose Up
-- One row per in-progress guest cart on an internal storefront channel
-- (domain/storefront.Cart). Token (not id) is what the storefront frontend
-- holds and presents on every call -- a random, unguessable bearer
-- capability for this cart, same reasoning as any other opaque session
-- token. Items is JSONB, same driver.Valuer/sql.Scanner + whole-row-replace
-- convention as whatsapp_conversations.cart (migration 137): a cart's line
-- set is small and always rewritten wholesale, never worth a child table.
CREATE TABLE storefront_carts (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    token            TEXT        NOT NULL,
    items            JSONB       NOT NULL DEFAULT '[]',
    checked_out_at   TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX storefront_carts_token_idx ON storefront_carts (token);
CREATE INDEX storefront_carts_tenant_id_idx ON storefront_carts (tenant_id);
CREATE INDEX storefront_carts_sales_channel_id_idx ON storefront_carts (sales_channel_id);

ALTER TABLE storefront_carts ENABLE ROW LEVEL SECURITY;
ALTER TABLE storefront_carts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON storefront_carts
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE storefront_carts;
