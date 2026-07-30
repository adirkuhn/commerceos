-- +goose Up

CREATE TABLE warehouses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE batches (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    sku_id       UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    warehouse_id UUID        NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    lot_number   TEXT        NOT NULL DEFAULT '',
    expiry_date  DATE,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX batches_sku_id_idx ON batches (sku_id);

-- Append-only ledger: no updated_at, rows are never modified after insert.
CREATE TABLE stock_movements (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    sku_id         UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    warehouse_id   UUID        NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    batch_id       UUID        REFERENCES batches(id) ON DELETE RESTRICT,
    type           TEXT        NOT NULL,
    quantity_delta BIGINT      NOT NULL,
    reason         TEXT        NOT NULL DEFAULT '',
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_movements_type_check CHECK (type IN ('receipt', 'sale', 'adjustment', 'transfer_out', 'transfer_in')),
    CONSTRAINT stock_movements_quantity_nonzero_check CHECK (quantity_delta <> 0)
);

CREATE INDEX stock_movements_sku_id_idx ON stock_movements (sku_id);
CREATE INDEX stock_movements_batch_id_idx ON stock_movements (batch_id);

ALTER TABLE warehouses      ENABLE ROW LEVEL SECURITY;
ALTER TABLE batches         ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON warehouses
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON batches
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

CREATE POLICY tenant_isolation ON stock_movements
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = current_setting('app.current_tenant', true)::uuid);

-- +goose Down

DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS batches;
DROP TABLE IF EXISTS warehouses;
