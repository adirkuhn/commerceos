-- +goose Up

CREATE TABLE product_channel_listings (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    product_id      UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    connection_id   UUID        NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    provider        TEXT        NOT NULL,
    external_id     TEXT        NOT NULL,
    sync_status     TEXT        NOT NULL DEFAULT 'pending',
    last_synced_at  TIMESTAMPTZ,
    last_sync_error TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_channel_listings_sync_status_check CHECK (sync_status IN ('pending', 'synced', 'error')),
    UNIQUE (connection_id, external_id)
);

CREATE INDEX product_channel_listings_product_id_idx ON product_channel_listings (product_id);

ALTER TABLE product_channel_listings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON product_channel_listings
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down

DROP TABLE IF EXISTS product_channel_listings;
