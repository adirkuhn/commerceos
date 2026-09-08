-- +goose Up
-- Shopper-submitted product reviews on the internal storefront
-- (domain/reviews.Review) -- one row per review, scoped to the sales
-- channel it was left on, same convention as storefront_carts (migration
-- 146).
CREATE TABLE reviews (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    sales_channel_id  UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    product_id        UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    customer_name     TEXT        NOT NULL,
    customer_email    TEXT        NOT NULL,
    rating            INT         NOT NULL,
    title             TEXT        NOT NULL DEFAULT '',
    body              TEXT        NOT NULL,
    verified          BOOLEAN     NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT reviews_rating_check CHECK (rating BETWEEN 1 AND 5)
);

CREATE INDEX reviews_tenant_id_idx ON reviews (tenant_id);
CREATE INDEX reviews_sales_channel_product_idx ON reviews (sales_channel_id, product_id);

ALTER TABLE reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE reviews FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON reviews
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE reviews;
