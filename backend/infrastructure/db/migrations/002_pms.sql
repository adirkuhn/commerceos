-- +goose Up

CREATE TABLE categories (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID        NOT NULL,
    parent_id UUID        REFERENCES categories(id) ON DELETE SET NULL,
    name      TEXT        NOT NULL,
    slug      TEXT        NOT NULL,
    position  INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug)
);

CREATE TABLE products (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    category_id UUID        REFERENCES categories(id) ON DELETE SET NULL,
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug),
    CONSTRAINT products_status_check CHECK (status IN ('draft', 'active', 'archived'))
);

CREATE TABLE attribute_definitions (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID        NOT NULL,
    name      TEXT        NOT NULL,
    type      TEXT        NOT NULL,
    options   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name),
    CONSTRAINT attribute_definitions_type_check CHECK (type IN ('text', 'number', 'boolean', 'select'))
);

CREATE TABLE variants (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    product_id UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    position   INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE variant_attributes (
    variant_id              UUID NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    attribute_definition_id UUID NOT NULL REFERENCES attribute_definitions(id),
    tenant_id               UUID NOT NULL,
    value                   TEXT NOT NULL,
    PRIMARY KEY (variant_id, attribute_definition_id)
);

CREATE TABLE skus (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    variant_id       UUID        NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    sku_code         TEXT        NOT NULL,
    barcode          TEXT,
    price_minor_units BIGINT     NOT NULL DEFAULT 0,
    currency_code    CHAR(3)     NOT NULL DEFAULT 'USD',
    status           TEXT        NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, sku_code),
    CONSTRAINT skus_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE media (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    product_id UUID        REFERENCES products(id) ON DELETE CASCADE,
    variant_id UUID        REFERENCES variants(id) ON DELETE CASCADE,
    url        TEXT        NOT NULL,
    type       TEXT        NOT NULL DEFAULT 'image',
    position   INT         NOT NULL DEFAULT 0,
    alt_text   TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_type_check CHECK (type IN ('image', 'video'))
);

-- Fine-grained RBAC: (user_id, tenant_id, rule)
-- Example rules: pms.product.create, pms.product.read, pms.sku.update
CREATE TABLE user_permissions (
    user_id   UUID NOT NULL,
    tenant_id UUID NOT NULL,
    rule      TEXT NOT NULL,
    PRIMARY KEY (user_id, tenant_id, rule)
);

-- ─── Row Level Security ───────────────────────────────────────────────────────

ALTER TABLE categories         ENABLE ROW LEVEL SECURITY;
ALTER TABLE products           ENABLE ROW LEVEL SECURITY;
ALTER TABLE attribute_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE variants           ENABLE ROW LEVEL SECURITY;
ALTER TABLE variant_attributes ENABLE ROW LEVEL SECURITY;
ALTER TABLE skus               ENABLE ROW LEVEL SECURITY;
ALTER TABLE media              ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_permissions   ENABLE ROW LEVEL SECURITY;

-- super_admin bypasses by setting app.current_tenant = 'bypass_rls'
CREATE POLICY tenant_isolation ON categories
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON products
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON attribute_definitions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON variants
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON variant_attributes
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON skus
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON media
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON user_permissions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down

DROP TABLE IF EXISTS user_permissions;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS skus;
DROP TABLE IF EXISTS variant_attributes;
DROP TABLE IF EXISTS variants;
DROP TABLE IF EXISTS attribute_definitions;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
