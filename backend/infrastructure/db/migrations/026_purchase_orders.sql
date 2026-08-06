-- +goose Up
CREATE TABLE purchase_orders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    supplier_id   UUID        NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    warehouse_id  UUID        NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    status        TEXT        NOT NULL DEFAULT 'draft',
    currency_code TEXT        NOT NULL DEFAULT 'USD',
    expected_date DATE,
    notes         TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_orders_status_check
        CHECK (status IN ('draft', 'sent', 'partially_received', 'received', 'cancelled', 'needs_review'))
);

CREATE INDEX purchase_orders_tenant_id_idx ON purchase_orders (tenant_id);
CREATE INDEX purchase_orders_supplier_id_idx ON purchase_orders (supplier_id);

CREATE TABLE purchase_order_line_items (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID        NOT NULL,
    purchase_order_id     UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    sku_id                UUID        REFERENCES skus(id) ON DELETE SET NULL,
    raw_description       TEXT        NOT NULL DEFAULT '',
    quantity              BIGINT      NOT NULL,
    unit_cost_minor_units BIGINT      NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_order_line_items_quantity_check CHECK (quantity > 0)
);

CREATE INDEX purchase_order_line_items_po_id_idx ON purchase_order_line_items (purchase_order_id);

ALTER TABLE purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders FORCE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_line_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_line_items FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON purchase_orders
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON purchase_order_line_items
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE purchase_order_line_items;
DROP TABLE purchase_orders;
