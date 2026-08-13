-- +goose Up
-- Minimal OMS foundation: a SalesOrder is a completed in-store sale pushed
-- in from an external source (today: POS) -- NOT a procurement.PurchaseOrder
-- (inbound supply from a supplier). Named sales_orders throughout,
-- deliberately distinct from purchase_orders, so the two are never
-- confused at the schema, domain, or API level.
-- external_order_id lets the source retry a push safely --
-- ReceivePOSSalesOrderUseCase is idempotent on
-- (tenant_id, source, external_order_id), so a network retry never creates
-- a duplicate sales order or double-decrements stock.
CREATE TABLE sales_orders (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    source              TEXT        NOT NULL,
    external_order_id   TEXT        NOT NULL,
    warehouse_id        UUID        NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    currency_code       TEXT        NOT NULL DEFAULT 'USD',
    total_minor_units   BIGINT      NOT NULL DEFAULT 0,
    occurred_at         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_orders_source_check CHECK (source IN ('pos')),
    CONSTRAINT sales_orders_tenant_source_external_id_unique UNIQUE (tenant_id, source, external_order_id)
);

CREATE INDEX sales_orders_tenant_id_idx ON sales_orders (tenant_id);

CREATE TABLE sales_order_line_items (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID        NOT NULL,
    sales_order_id           UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    sku_id                   UUID        NOT NULL REFERENCES skus(id) ON DELETE RESTRICT,
    quantity                 BIGINT      NOT NULL,
    unit_price_minor_units   BIGINT      NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_order_line_items_quantity_check CHECK (quantity > 0)
);

CREATE INDEX sales_order_line_items_sales_order_id_idx ON sales_order_line_items (sales_order_id);

ALTER TABLE sales_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_orders FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_order_line_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_line_items FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_orders
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON sales_order_line_items
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE sales_order_line_items;
DROP TABLE sales_orders;
