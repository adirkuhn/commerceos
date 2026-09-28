-- +goose Up
-- Merchandising collections (domain/collections). Smart collections keep
-- their rules and are evaluated on read; manual ones list their products
-- in collection_products.
CREATE TABLE collections (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            TEXT        NOT NULL,
    slug            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    image_asset_id  UUID,
    seo_title       TEXT        NOT NULL DEFAULT '',
    seo_description TEXT        NOT NULL DEFAULT '',
    type            TEXT        NOT NULL CHECK (type IN ('manual', 'smart')),
    match           TEXT        NOT NULL DEFAULT 'all' CHECK (match IN ('all', 'any')),
    rules           JSONB       NOT NULL DEFAULT '[]',
    sort            TEXT        NOT NULL CHECK (sort IN ('manual', 'name_asc', 'price_asc', 'price_desc', 'newest')),
    published       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT collections_tenant_slug_unique UNIQUE (tenant_id, slug)
);

CREATE TABLE collection_products (
    tenant_id     UUID NOT NULL,
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    product_id    UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    position      INT  NOT NULL,
    PRIMARY KEY (collection_id, product_id)
);
CREATE INDEX collection_products_product_idx ON collection_products (product_id);

ALTER TABLE collections ENABLE ROW LEVEL SECURITY;
ALTER TABLE collections FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON collections
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE collection_products ENABLE ROW LEVEL SECURITY;
ALTER TABLE collection_products FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON collection_products
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE collection_products;
DROP TABLE collections;
