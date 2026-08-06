-- +goose Up
CREATE TABLE purchase_order_documents (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    purchase_order_id UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    file_url          TEXT        NOT NULL,
    file_name         TEXT        NOT NULL DEFAULT '',
    content_type      TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX purchase_order_documents_tenant_id_idx ON purchase_order_documents (tenant_id);
CREATE INDEX purchase_order_documents_po_id_idx ON purchase_order_documents (purchase_order_id);

CREATE TABLE document_parsing_jobs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    purchase_order_id UUID        NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    document_id       UUID        NOT NULL REFERENCES purchase_order_documents(id) ON DELETE CASCADE,
    status            TEXT        NOT NULL DEFAULT 'pending',
    error_message     TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ,
    CONSTRAINT document_parsing_jobs_status_check
        CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

CREATE INDEX document_parsing_jobs_tenant_id_idx ON document_parsing_jobs (tenant_id);
CREATE INDEX document_parsing_jobs_status_idx ON document_parsing_jobs (status);

CREATE TABLE supplier_product_aliases (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    supplier_id UUID        NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    raw_key     TEXT        NOT NULL,
    sku_id      UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_product_aliases_unique UNIQUE (tenant_id, supplier_id, raw_key)
);

CREATE INDEX supplier_product_aliases_tenant_id_idx ON supplier_product_aliases (tenant_id);

ALTER TABLE purchase_order_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_order_documents FORCE ROW LEVEL SECURITY;
ALTER TABLE document_parsing_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_parsing_jobs FORCE ROW LEVEL SECURITY;
ALTER TABLE supplier_product_aliases ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_product_aliases FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON purchase_order_documents
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON document_parsing_jobs
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE POLICY tenant_isolation ON supplier_product_aliases
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE supplier_product_aliases;
DROP TABLE document_parsing_jobs;
DROP TABLE purchase_order_documents;
