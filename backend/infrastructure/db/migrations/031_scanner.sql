-- +goose Up
CREATE TABLE scanner_pairing_codes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    code       TEXT        NOT NULL,
    user_id    TEXT        NOT NULL,
    role       TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at TIMESTAMPTZ,
    CONSTRAINT scanner_pairing_codes_status_check CHECK (status IN ('pending', 'claimed'))
);

CREATE INDEX scanner_pairing_codes_tenant_id_idx ON scanner_pairing_codes (tenant_id);
-- Only pending codes need to be globally unique -- the claim endpoint has
-- no tenant identity yet, so it looks a code up by text alone.
CREATE UNIQUE INDEX scanner_pairing_codes_pending_code_idx ON scanner_pairing_codes (code) WHERE status = 'pending';

CREATE TABLE bookings (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    purchase_order_id UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    status            TEXT        NOT NULL DEFAULT 'in_progress',
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paused_at         TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    completed_by      TEXT        NOT NULL DEFAULT '',
    CONSTRAINT bookings_status_check CHECK (status IN ('in_progress', 'paused', 'completed'))
);

CREATE INDEX bookings_tenant_id_idx ON bookings (tenant_id);
CREATE INDEX bookings_po_id_idx ON bookings (purchase_order_id);

CREATE TABLE booking_scans (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    booking_id UUID        NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    sku_id     UUID        NOT NULL REFERENCES skus(id) ON DELETE RESTRICT,
    batch_id   UUID        REFERENCES batches(id) ON DELETE SET NULL,
    quantity   BIGINT      NOT NULL,
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scanned_by TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX booking_scans_tenant_id_idx ON booking_scans (tenant_id);
CREATE INDEX booking_scans_booking_id_idx ON booking_scans (booking_id);

CREATE TABLE barcode_aliases (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    sku_id         UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    code           TEXT        NOT NULL,
    units_per_scan BIGINT      NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);

CREATE INDEX barcode_aliases_tenant_id_idx ON barcode_aliases (tenant_id);
CREATE INDEX barcode_aliases_sku_id_idx ON barcode_aliases (sku_id);

-- The scanner app looks SKUs up by unit barcode frequently; this column had
-- no index before since no read path needed one.
CREATE INDEX skus_tenant_barcode_idx ON skus (tenant_id, barcode) WHERE barcode IS NOT NULL AND barcode <> '';

ALTER TABLE scanner_pairing_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE scanner_pairing_codes FORCE ROW LEVEL SECURITY;
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE bookings FORCE ROW LEVEL SECURITY;
ALTER TABLE booking_scans ENABLE ROW LEVEL SECURITY;
ALTER TABLE booking_scans FORCE ROW LEVEL SECURITY;
ALTER TABLE barcode_aliases ENABLE ROW LEVEL SECURITY;
ALTER TABLE barcode_aliases FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON scanner_pairing_codes
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON bookings
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON booking_scans
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON barcode_aliases
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP INDEX skus_tenant_barcode_idx;
DROP TABLE barcode_aliases;
DROP TABLE booking_scans;
DROP TABLE bookings;
DROP TABLE scanner_pairing_codes;
