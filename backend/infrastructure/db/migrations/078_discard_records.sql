-- +goose Up

-- DiscardRecord is the audit trail for "this stock was thrown away" --
-- expired/near-expiry write-off (desktop review queue or the scanner's
-- Discard app), with an optional photo. movement_id always points at the
-- Lost StockMovement that actually removed the stock -- this table is the
-- richer, queryable report row on top of that, not a replacement for the
-- ledger. See domain/inventory/discard.go.
CREATE TABLE discard_records (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    sku_id        UUID        NOT NULL REFERENCES skus(id) ON DELETE RESTRICT,
    warehouse_id  UUID        NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    batch_id      UUID        REFERENCES batches(id) ON DELETE RESTRICT,
    movement_id   UUID        NOT NULL REFERENCES stock_movements(id) ON DELETE RESTRICT,
    quantity      BIGINT      NOT NULL,
    reason        TEXT        NOT NULL DEFAULT '',
    photo_url     TEXT,
    discarded_by  TEXT        NOT NULL DEFAULT '',
    discarded_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT discard_records_quantity_check CHECK (quantity > 0)
);

CREATE INDEX discard_records_tenant_id_discarded_at_idx ON discard_records (tenant_id, discarded_at DESC);

ALTER TABLE discard_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE discard_records FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON discard_records
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE discard_records;
