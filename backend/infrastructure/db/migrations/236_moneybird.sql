-- +goose Up
-- Moneybird bookkeeping integration (provider "moneybird"): the API token
-- lives in integration_connections.consumer_secret_encrypted like every
-- other provider; everything Moneybird-specific lives here.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.

-- One row per Moneybird connection: which administration to book into,
-- the default ledger account, the VAT-rate and payment-method mappings,
-- and whether quotes/orders sync automatically (off: manual send only).
CREATE TABLE moneybird_settings (
    connection_id            UUID        PRIMARY KEY REFERENCES integration_connections(id) ON DELETE CASCADE,
    tenant_id                UUID        NOT NULL,
    administration_id        TEXT        NOT NULL DEFAULT '',
    administration_name      TEXT        NOT NULL DEFAULT '',
    default_ledger_account_id TEXT       NOT NULL DEFAULT '',
    guest_contact_id         TEXT        NOT NULL DEFAULT '',
    auto_sync                BOOLEAN     NOT NULL DEFAULT FALSE,
    -- [{"rate_basis_points":2100,"moneybird_tax_rate_id":"..."}]
    tax_rates                JSONB       NOT NULL DEFAULT '[]',
    -- [{"method":"bacs","moneybird_payment_method":"bank_transfer"}]
    payment_methods          JSONB       NOT NULL DEFAULT '[]',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The Moneybird document a quote (estimate) or sales order (external sales
-- invoice) was sent as -- at most one per source, so a document is never
-- created twice.
CREATE TABLE moneybird_documents (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL,
    connection_id      UUID        NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    source_type        TEXT        NOT NULL CHECK (source_type IN ('quote', 'sales_order')),
    source_id          UUID        NOT NULL,
    moneybird_id       TEXT        NOT NULL DEFAULT '',
    reference          TEXT        NOT NULL DEFAULT '',
    status             TEXT        NOT NULL CHECK (status IN ('synced', 'failed')),
    payment_registered BOOLEAN     NOT NULL DEFAULT FALSE,
    error              TEXT        NOT NULL DEFAULT '',
    synced_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, source_type, source_id)
);

-- A customer's Moneybird contact, found or created on first send.
CREATE TABLE moneybird_contacts (
    tenant_id            UUID        NOT NULL,
    connection_id        UUID        NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    customer_id          UUID        NOT NULL,
    moneybird_contact_id TEXT        NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, connection_id, customer_id)
);

ALTER TABLE moneybird_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE moneybird_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON moneybird_settings
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE moneybird_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE moneybird_documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON moneybird_documents
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE moneybird_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE moneybird_contacts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON moneybird_contacts
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS moneybird_contacts;
DROP TABLE IF EXISTS moneybird_documents;
DROP TABLE IF EXISTS moneybird_settings;
