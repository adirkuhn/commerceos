-- +goose Up
CREATE TABLE sales_channels (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID        NOT NULL,
    name                   TEXT        NOT NULL,
    type                   TEXT        NOT NULL,
    connection_id          UUID        REFERENCES integration_connections(id) ON DELETE CASCADE,
    enrollment_mode        TEXT        NOT NULL DEFAULT 'manual',
    enrollment_category_id UUID        REFERENCES categories(id) ON DELETE SET NULL,
    enrollment_tag         TEXT        NOT NULL DEFAULT '',
    active                 BOOLEAN     NOT NULL DEFAULT true,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_channels_type_check CHECK (type IN ('woocommerce', 'internal')),
    CONSTRAINT sales_channels_enrollment_mode_check CHECK (enrollment_mode IN ('all', 'category', 'tag', 'manual'))
);

CREATE INDEX sales_channels_tenant_id_idx ON sales_channels (tenant_id);

CREATE TABLE sales_channel_product_overrides (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    channel_id UUID        NOT NULL REFERENCES sales_channels(id) ON DELETE CASCADE,
    product_id UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    included   BOOLEAN     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_channel_product_overrides_unique UNIQUE (channel_id, product_id)
);

CREATE INDEX sales_channel_product_overrides_tenant_id_idx ON sales_channel_product_overrides (tenant_id);
CREATE INDEX sales_channel_product_overrides_channel_id_idx ON sales_channel_product_overrides (channel_id);

ALTER TABLE sales_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_channels FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_channel_product_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_channel_product_overrides FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_channels
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON sales_channel_product_overrides
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE sales_channel_product_overrides;
DROP TABLE sales_channels;
