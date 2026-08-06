-- +goose Up
CREATE TABLE receiving_records (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    purchase_order_id UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    invoice_file_url  TEXT        NOT NULL,
    invoice_file_name TEXT        NOT NULL DEFAULT '',
    content_type      TEXT        NOT NULL DEFAULT '',
    status            TEXT        NOT NULL DEFAULT 'pending_review',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT receiving_records_status_check CHECK (status IN ('pending_review', 'applied'))
);

CREATE INDEX receiving_records_tenant_id_idx ON receiving_records (tenant_id);
CREATE INDEX receiving_records_po_id_idx ON receiving_records (purchase_order_id);

CREATE TABLE receiving_parsing_jobs (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    receiving_record_id UUID        NOT NULL REFERENCES receiving_records(id) ON DELETE CASCADE,
    status              TEXT        NOT NULL DEFAULT 'pending',
    error_message       TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at          TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ,
    CONSTRAINT receiving_parsing_jobs_status_check
        CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

CREATE INDEX receiving_parsing_jobs_tenant_id_idx ON receiving_parsing_jobs (tenant_id);
CREATE INDEX receiving_parsing_jobs_status_idx ON receiving_parsing_jobs (status);

CREATE TABLE receiving_discrepancies (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    receiving_record_id UUID        NOT NULL REFERENCES receiving_records(id) ON DELETE CASCADE,
    line_item_id        UUID        REFERENCES purchase_order_line_items(id) ON DELETE SET NULL,
    type                TEXT        NOT NULL,
    expected_qty        BIGINT      NOT NULL DEFAULT 0,
    received_qty        BIGINT      NOT NULL DEFAULT 0,
    details             TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT receiving_discrepancies_type_check
        CHECK (type IN ('quantity_mismatch', 'missing_item', 'unexpected_item', 'expiry_mismatch'))
);

CREATE INDEX receiving_discrepancies_tenant_id_idx ON receiving_discrepancies (tenant_id);
CREATE INDEX receiving_discrepancies_record_id_idx ON receiving_discrepancies (receiving_record_id);

ALTER TABLE receiving_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE receiving_records FORCE ROW LEVEL SECURITY;
ALTER TABLE receiving_parsing_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE receiving_parsing_jobs FORCE ROW LEVEL SECURITY;
ALTER TABLE receiving_discrepancies ENABLE ROW LEVEL SECURITY;
ALTER TABLE receiving_discrepancies FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON receiving_records
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON receiving_parsing_jobs
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON receiving_discrepancies
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE receiving_discrepancies;
DROP TABLE receiving_parsing_jobs;
DROP TABLE receiving_records;
