-- +goose Up

-- Packed is Fulfilled's packing-phase counterpart -- see
-- domain/oms/sales_order.go's doc comment. Backfilled true for every
-- already-fulfilled order (packing/shipping is assumed done for history
-- that predates this feature); every new order starts unpacked.
ALTER TABLE sales_orders ADD COLUMN packed BOOLEAN NOT NULL DEFAULT false;
UPDATE sales_orders SET packed = true WHERE fulfilled = true;

CREATE TABLE packing_sessions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    sales_order_id UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    status         TEXT        NOT NULL DEFAULT 'in_progress',
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ,
    completed_by   TEXT        NOT NULL DEFAULT '',
    locked_by      TEXT,
    locked_at      TIMESTAMPTZ,
    CONSTRAINT packing_sessions_status_check CHECK (status IN ('in_progress', 'completed'))
);

CREATE INDEX packing_sessions_tenant_id_idx ON packing_sessions (tenant_id);
CREATE INDEX packing_sessions_sales_order_id_idx ON packing_sessions (sales_order_id);

-- PackScan is NOT staged the way pick_scans is -- packing never touches
-- stock or the order's line items, so a row is real the moment it's
-- created. See domain/oms/packing.go's doc comment.
CREATE TABLE pack_scans (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    session_id    UUID        NOT NULL REFERENCES packing_sessions(id) ON DELETE CASCADE,
    line_item_id  UUID        NOT NULL REFERENCES sales_order_line_items(id) ON DELETE CASCADE,
    sku_id        UUID        NOT NULL REFERENCES skus(id) ON DELETE RESTRICT,
    quantity      BIGINT      NOT NULL,
    scanned_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scanned_by    TEXT        NOT NULL DEFAULT '',
    CONSTRAINT pack_scans_quantity_check CHECK (quantity > 0)
);

CREATE INDEX pack_scans_tenant_id_idx ON pack_scans (tenant_id);
CREATE INDEX pack_scans_session_id_idx ON pack_scans (session_id);

ALTER TABLE packing_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE packing_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE pack_scans ENABLE ROW LEVEL SECURITY;
ALTER TABLE pack_scans FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON packing_sessions
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON pack_scans
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE pack_scans;
DROP TABLE packing_sessions;
ALTER TABLE sales_orders DROP COLUMN packed;
