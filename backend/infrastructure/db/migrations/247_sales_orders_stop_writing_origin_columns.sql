-- +goose Up
-- COM-15: this release no longer writes sales_orders.source,
-- external_order_id, connection_id or order_key -- the order's origin row
-- (sales_order_origins, migration 246) holds them. The previous release
-- still reads and writes them, so for one release they stay filled:
--
-- * defaults let an insert without them pass the NOT NULL / CHECK
--   constraints;
-- * a trigger copies every saved origin back onto its order.
--
-- Dropped, with migration 246's trigger, in the release that drops the
-- columns.

ALTER TABLE sales_orders ALTER COLUMN source SET DEFAULT 'manual';
ALTER TABLE sales_orders ALTER COLUMN external_order_id SET DEFAULT gen_random_uuid()::text;

-- Per connection now (two shops may both have order #1001). The name stays:
-- the previous release maps exactly this constraint to "lost a create race".
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_tenant_source_external_id_unique;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_tenant_source_external_id_unique
    UNIQUE NULLS NOT DISTINCT (tenant_id, source, connection_id, external_order_id);

-- An order with no number showed its external id instead; that id now
-- lives on the origin, so the number takes it.
UPDATE sales_orders SET number = external_order_id
WHERE number = '' AND source IN ('pos', 'woocommerce', 'bol');

-- Both triggers write the other table; each skips when fired by the other
-- (pg_trigger_depth() > 1), so neither recurses.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION sales_order_origin_from_order() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN NEW;
    END IF;
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

-- +goose StatementBegin
CREATE FUNCTION sales_order_columns_from_origin() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN NEW;
    END IF;
    UPDATE sales_orders SET
        source = NEW.origin,
        external_order_id = COALESCE(NEW.external_order_id, sales_orders.external_order_id),
        connection_id = NEW.connection_id,
        order_key = COALESCE(NEW.external_key, '')
    WHERE id = NEW.sales_order_id;
    RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER sales_order_columns_from_origin
    AFTER INSERT OR UPDATE ON sales_order_origins
    FOR EACH ROW EXECUTE FUNCTION sales_order_columns_from_origin();

-- +goose Down
DROP TRIGGER sales_order_columns_from_origin ON sales_order_origins;
DROP FUNCTION sales_order_columns_from_origin();
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION sales_order_origin_from_order() RETURNS trigger LANGUAGE plpgsql AS $$
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
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_tenant_source_external_id_unique;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_tenant_source_external_id_unique UNIQUE (tenant_id, source, external_order_id);
ALTER TABLE sales_orders ALTER COLUMN external_order_id DROP DEFAULT;
ALTER TABLE sales_orders ALTER COLUMN source DROP DEFAULT;
