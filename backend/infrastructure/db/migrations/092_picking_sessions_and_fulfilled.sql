-- +goose Up

-- Fulfilled is a binary secondary status on SalesOrder, independent of
-- Status -- see domain/oms/sales_order.go's doc comment. Backfilled true
-- for every existing completed order; every other existing order (and
-- every new one) starts unfulfilled and is flipped explicitly by
-- CompletePickingUseCase, or automatically the moment Status transitions
-- to completed via any path (UpdateSalesOrderUseCase, WooCommerce
-- import/sync).
ALTER TABLE sales_orders ADD COLUMN fulfilled BOOLEAN NOT NULL DEFAULT false;
UPDATE sales_orders SET fulfilled = true WHERE status = 'completed';

CREATE TABLE picking_sessions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    sales_order_id UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    status         TEXT        NOT NULL DEFAULT 'in_progress',
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paused_at      TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    completed_by   TEXT        NOT NULL DEFAULT '',
    CONSTRAINT picking_sessions_status_check CHECK (status IN ('in_progress', 'paused', 'completed'))
);

CREATE INDEX picking_sessions_tenant_id_idx ON picking_sessions (tenant_id);
CREATE INDEX picking_sessions_sales_order_id_idx ON picking_sessions (sales_order_id);

-- PickScan is a STAGED action -- see domain/oms/picking.go's doc comment,
-- same "nothing real until CompletePickingUseCase replays it" philosophy
-- as procurement.booking_scans. line_item_id is the SalesOrderLineItem this
-- scan targets: for 'pick' it's the line being confirmed; for 'substitute'
-- it's the ORIGINAL line being replaced; nil for 'freebie'/'new_item'
-- (nothing on the order yet to reference). sku_id is the line's own SKU
-- for 'pick', the SUBSTITUTE sku for 'substitute', or the new SKU being
-- added for 'freebie'/'new_item'. price_minor_units is nil for
-- pick/substitute (no real line item created), 0 for freebie, picker-
-- entered for new_item.
CREATE TABLE pick_scans (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    session_id        UUID        NOT NULL REFERENCES picking_sessions(id) ON DELETE CASCADE,
    line_item_id      UUID        REFERENCES sales_order_line_items(id) ON DELETE SET NULL,
    sku_id            UUID        NOT NULL REFERENCES skus(id) ON DELETE RESTRICT,
    action_type       TEXT        NOT NULL DEFAULT 'pick',
    quantity          BIGINT      NOT NULL,
    price_minor_units BIGINT,
    reason            TEXT        NOT NULL DEFAULT '',
    scanned_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scanned_by        TEXT        NOT NULL DEFAULT '',
    replayed_at       TIMESTAMPTZ,
    CONSTRAINT pick_scans_action_type_check CHECK (action_type IN ('pick', 'substitute', 'freebie', 'new_item')),
    CONSTRAINT pick_scans_quantity_check CHECK (quantity > 0)
);

CREATE INDEX pick_scans_tenant_id_idx ON pick_scans (tenant_id);
CREATE INDEX pick_scans_session_id_idx ON pick_scans (session_id);

ALTER TABLE picking_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE picking_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE pick_scans ENABLE ROW LEVEL SECURITY;
ALTER TABLE pick_scans FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON picking_sessions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON pick_scans
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE pick_scans;
DROP TABLE picking_sessions;
ALTER TABLE sales_orders DROP COLUMN fulfilled;
