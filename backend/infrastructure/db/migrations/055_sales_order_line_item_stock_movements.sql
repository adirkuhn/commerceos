-- +goose Up
-- Links a sales_order_line_item to the stock_movement(s) it caused --
-- stock_movements.reason was only ever a free-text string, so there was no
-- reliable way to ask "which movements did this sale actually produce"
-- (a decrement on creation, later a restock on cancel, are separate rows).
-- A dedicated link table rather than a nullable FK column on
-- stock_movements: that table is shared by every movement source
-- (receiving, booking scans, splits, sales) and a growing set of
-- source-specific nullable FKs there does not scale the way this does.
CREATE TABLE sales_order_line_item_stock_movements (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    sales_order_line_item_id  UUID        NOT NULL REFERENCES sales_order_line_items(id) ON DELETE CASCADE,
    stock_movement_id         UUID        NOT NULL REFERENCES stock_movements(id) ON DELETE CASCADE,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_order_line_item_stock_movements_movement_unique UNIQUE (stock_movement_id)
);

CREATE INDEX sales_order_line_item_stock_movements_line_item_id_idx
    ON sales_order_line_item_stock_movements (sales_order_line_item_id);

ALTER TABLE sales_order_line_item_stock_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_line_item_stock_movements FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_order_line_item_stock_movements
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE sales_order_line_item_stock_movements;
