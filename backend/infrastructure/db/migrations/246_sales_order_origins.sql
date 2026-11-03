-- +goose Up
-- COM-15: a sales order holds order data only. Where it came from (origin)
-- and, for an imported order, which external order it is, move to one row
-- per order here -- generic across integrations, a new one adds an origin
-- value, never a column. The sales_orders columns (source,
-- external_order_id, order_key, connection_id) stay until a later release
-- so the previous one still runs.
CREATE TABLE sales_order_origins (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    sales_order_id    UUID        NOT NULL UNIQUE REFERENCES sales_orders(id) ON DELETE CASCADE,
    origin            TEXT        NOT NULL CHECK (origin IN ('pos', 'manual', 'woocommerce', 'bol', 'whatsapp', 'internal')),
    -- RESTRICT: connections are never hard-deleted any more (migration
    -- 245); losing this link silently is what caused the incident.
    connection_id     UUID        REFERENCES integration_connections(id) ON DELETE RESTRICT,
    external_order_id TEXT,
    external_key      TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per connection, so order #1001 from two Woo shops are two orders (the old
-- sales_orders (tenant_id, source, external_order_id) rule rejected that).
CREATE UNIQUE INDEX sales_order_origins_connection_external_idx
    ON sales_order_origins (tenant_id, connection_id, external_order_id)
    WHERE connection_id IS NOT NULL;
-- Systems without a connection row (POS): unique per origin.
CREATE UNIQUE INDEX sales_order_origins_origin_external_idx
    ON sales_order_origins (tenant_id, origin, external_order_id)
    WHERE connection_id IS NULL AND external_order_id IS NOT NULL;
CREATE INDEX sales_order_origins_connection_idx ON sales_order_origins (connection_id);

ALTER TABLE sales_order_origins ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_origins FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_order_origins
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Keeps the origin in step with whatever still writes the old sales_orders
-- columns: the previous release (during a deploy, or after a rollback)
-- and this one until its writers move over. Only external references are
-- copied: an order made here has a random UUID there, not an external id.
-- Dropped with the old columns.
-- +goose StatementBegin
CREATE FUNCTION sales_order_origin_from_order() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO sales_order_origins (tenant_id, sales_order_id, origin, connection_id, external_order_id, external_key)
    VALUES (
        NEW.tenant_id, NEW.id, NEW.source,
        CASE WHEN NEW.source IN ('woocommerce', 'bol') THEN NEW.connection_id END,
        CASE WHEN NEW.source IN ('pos', 'woocommerce', 'bol') THEN NULLIF(NEW.external_order_id, '') END,
        NULLIF(NEW.order_key, ''))
    ON CONFLICT (sales_order_id) DO UPDATE SET
        origin = EXCLUDED.origin,
        connection_id = COALESCE(EXCLUDED.connection_id, sales_order_origins.connection_id),
        external_order_id = EXCLUDED.external_order_id,
        external_key = EXCLUDED.external_key;
    RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER sales_order_origin_from_order
    AFTER INSERT OR UPDATE OF source, external_order_id, order_key, connection_id ON sales_orders
    FOR EACH ROW WHEN (NEW.source IS NOT NULL AND NEW.source <> '')
    EXECUTE FUNCTION sales_order_origin_from_order();

-- Backfill every existing order. A shop order whose connection was lost
-- (the 2026-11-03 SET NULL) takes the tenant's only connection of that
-- provider; with none or several it stays NULL and is reported below.
INSERT INTO sales_order_origins (tenant_id, sales_order_id, origin, connection_id, external_order_id, external_key, created_at)
SELECT so.tenant_id, so.id, so.source,
    CASE WHEN so.source IN ('woocommerce', 'bol') THEN COALESCE(so.connection_id, only_conn.id) END,
    CASE WHEN so.source IN ('pos', 'woocommerce', 'bol') THEN NULLIF(so.external_order_id, '') END,
    NULLIF(so.order_key, ''),
    so.created_at
FROM sales_orders so
LEFT JOIN (
    -- MIN() has no uuid overload; only one row per group survives HAVING.
    SELECT tenant_id, provider, MIN(id::text)::uuid AS id
    FROM integration_connections
    GROUP BY tenant_id, provider
    HAVING COUNT(*) = 1
) only_conn ON only_conn.tenant_id = so.tenant_id AND only_conn.provider = so.source
ON CONFLICT (sales_order_id) DO NOTHING;

-- +goose StatementBegin
DO $$
DECLARE unmapped BIGINT;
BEGIN
    SELECT count(*) INTO unmapped FROM sales_order_origins
    WHERE origin IN ('woocommerce', 'bol') AND connection_id IS NULL;
    IF unmapped > 0 THEN
        RAISE NOTICE 'sales_order_origins: % shop orders have no connection (kept, origin set): %', unmapped,
            (SELECT string_agg(sales_order_id::text, ', ') FROM (
                SELECT sales_order_id FROM sales_order_origins
                WHERE origin IN ('woocommerce', 'bol') AND connection_id IS NULL
                ORDER BY sales_order_id LIMIT 50) s);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER sales_order_origin_from_order ON sales_orders;
DROP FUNCTION sales_order_origin_from_order();
DROP TABLE sales_order_origins;
