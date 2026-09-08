-- +goose Up
-- One canvas document per (sales_channel_id, slug) -- "home" is the only
-- slug used today (see domain/theme's package doc comment). blocks is
-- JSONB, same whole-document-rewritten-wholesale convention as
-- whatsapp_conversations.cart (migration 137) and storefront_carts.items
-- (migration 146): a page's block list is always saved as one unit from
-- the canvas editor, never worth normalizing into child rows.
CREATE TABLE theme_pages (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    sales_channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    slug             TEXT        NOT NULL,
    canvas_width     INT         NOT NULL DEFAULT 1440,
    blocks           JSONB       NOT NULL DEFAULT '[]',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT theme_pages_channel_slug_unique UNIQUE (sales_channel_id, slug)
);

CREATE INDEX theme_pages_tenant_id_idx ON theme_pages (tenant_id);

ALTER TABLE theme_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE theme_pages FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON theme_pages
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE theme_pages;
