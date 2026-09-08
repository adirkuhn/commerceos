-- +goose Up
-- Tenant-wide cart-level discount codes -- deliberately its own table, not
-- folded into promotions (that module's own doc comment scopes coupons
-- out entirely). Not channel/product-scoped -- one code applies to any
-- cart on any internal storefront for the tenant.
CREATE TABLE coupons (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    code                      TEXT        NOT NULL,
    discount_type             TEXT        NOT NULL,
    discount_value            INT         NOT NULL,
    min_subtotal_minor_units  BIGINT,
    usage_limit               INT,
    times_used                INT         NOT NULL DEFAULT 0,
    expires_at                TIMESTAMPTZ,
    active                    BOOLEAN     NOT NULL DEFAULT true,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT coupons_discount_type_check CHECK (discount_type IN ('percent', 'fixed')),
    CONSTRAINT coupons_tenant_code_unique UNIQUE (tenant_id, code)
);

CREATE INDEX coupons_tenant_id_idx ON coupons (tenant_id);

ALTER TABLE coupons ENABLE ROW LEVEL SECURITY;
ALTER TABLE coupons FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON coupons
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE coupons;
