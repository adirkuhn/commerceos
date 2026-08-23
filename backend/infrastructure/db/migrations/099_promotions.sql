-- +goose Up

-- A named, time-boxed discount campaign -- tracked separately from
-- skus.price_minor_units/sale_price_minor_units (see the Phase 0 roadmap's
-- distinction between "what a SKU costs" and "why/how it's discounted").
-- v1 supports only two discount mechanisms (percentage, fixed_amount);
-- ends_at is nullable for an open-ended promotion.
CREATE TABLE promotions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    name       TEXT        NOT NULL,
    type       TEXT        NOT NULL,
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ,
    status     TEXT        NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX promotions_tenant_id_idx ON promotions (tenant_id);

-- One SKU's discount within a promotion. discount_type is independent of
-- the parent promotion's own type (see domain/promotions.PromotionSKU) so
-- lines can in principle mix mechanisms; discount_value is basis points for
-- percentage, minor currency units for fixed_amount.
CREATE TABLE promotion_skus (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    promotion_id   UUID        NOT NULL REFERENCES promotions(id) ON DELETE CASCADE,
    sku_id         UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    discount_type  TEXT        NOT NULL,
    discount_value BIGINT      NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT promotion_skus_promotion_sku_unique UNIQUE (promotion_id, sku_id)
);

CREATE INDEX promotion_skus_tenant_id_idx ON promotion_skus (tenant_id);
CREATE INDEX promotion_skus_promotion_id_idx ON promotion_skus (promotion_id);

ALTER TABLE promotions ENABLE ROW LEVEL SECURITY;
ALTER TABLE promotions FORCE ROW LEVEL SECURITY;
ALTER TABLE promotion_skus ENABLE ROW LEVEL SECURITY;
ALTER TABLE promotion_skus FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON promotions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON promotion_skus
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE promotion_skus;
DROP TABLE promotions;
