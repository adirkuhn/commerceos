-- +goose Up
-- Saved B2B quotes (domain/quotes). Line prices are stored as offered --
-- reopening a quote shows what was quoted, not today's price. sku_id and
-- product_id are not foreign keys on purpose: a quote is a record of what
-- was offered and stays when a product is later deleted (the screen shows
-- such a line as no longer available).
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE quotes (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID        NOT NULL,
    number                BIGINT      NOT NULL,
    name                  TEXT        NOT NULL CHECK (length(btrim(name)) > 0),
    discount_basis_points INTEGER     NOT NULL DEFAULT 0 CHECK (discount_basis_points BETWEEN 0 AND 10000),
    currency_code         TEXT        NOT NULL DEFAULT '',
    created_by            TEXT        NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, number)
);

CREATE INDEX quotes_tenant_updated_idx ON quotes (tenant_id, updated_at DESC);

CREATE TABLE quote_lines (
    quote_id               UUID    NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    tenant_id              UUID    NOT NULL,
    position               INTEGER NOT NULL,
    product_id             UUID    NOT NULL,
    sku_id                 UUID    NOT NULL,
    quantity               BIGINT  NOT NULL CHECK (quantity >= 0),
    unit_price_minor_units BIGINT  NOT NULL CHECK (unit_price_minor_units >= 0),
    PRIMARY KEY (quote_id, sku_id)
);

-- One row per tenant: the last quote number handed out. Numbers are never
-- reused, even after the newest quote is deleted.
CREATE TABLE quote_number_counters (
    tenant_id   UUID   PRIMARY KEY,
    last_number BIGINT NOT NULL
);

ALTER TABLE quotes ENABLE ROW LEVEL SECURITY;
ALTER TABLE quotes FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON quotes
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE quote_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE quote_lines FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON quote_lines
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

ALTER TABLE quote_number_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE quote_number_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON quote_number_counters
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS quote_number_counters;
DROP TABLE IF EXISTS quote_lines;
DROP TABLE IF EXISTS quotes;
