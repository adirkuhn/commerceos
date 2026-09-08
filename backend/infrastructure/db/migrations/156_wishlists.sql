-- +goose Up
-- One row per guest wishlist on an internal storefront channel
-- (domain/storefront.Wishlist) -- same token-addressed identity model as
-- storefront_carts (migration 146), but holds only product ids (JSONB
-- array of strings), never a price/name snapshot -- see Wishlist's own
-- doc comment on why.
CREATE TABLE storefront_wishlists (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    token            TEXT        NOT NULL,
    product_ids      JSONB       NOT NULL DEFAULT '[]',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX storefront_wishlists_token_idx ON storefront_wishlists (token);
CREATE INDEX storefront_wishlists_tenant_id_idx ON storefront_wishlists (tenant_id);

ALTER TABLE storefront_wishlists ENABLE ROW LEVEL SECURITY;
ALTER TABLE storefront_wishlists FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON storefront_wishlists
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE storefront_wishlists;
