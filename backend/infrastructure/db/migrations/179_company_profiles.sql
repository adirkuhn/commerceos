-- +goose Up

-- The tenant's own business identity: the seller block on invoices (legal
-- name, VAT, chamber of commerce, address, IBAN) and sender details.
-- One row per tenant; see domain/settings/company_profile.go.
CREATE TABLE company_profiles (
    tenant_id            UUID        PRIMARY KEY,
    legal_name           TEXT        NOT NULL,
    trading_name         TEXT        NOT NULL DEFAULT '',
    vat_number           TEXT        NOT NULL DEFAULT '',
    chamber_of_commerce  TEXT        NOT NULL DEFAULT '',
    email                TEXT        NOT NULL DEFAULT '',
    phone                TEXT        NOT NULL DEFAULT '',
    address_street       TEXT        NOT NULL DEFAULT '',
    address_house_number TEXT        NOT NULL DEFAULT '',
    address_postal_code  TEXT        NOT NULL DEFAULT '',
    address_city         TEXT        NOT NULL DEFAULT '',
    address_country      TEXT        NOT NULL DEFAULT '',
    iban                 TEXT        NOT NULL DEFAULT '',
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE company_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE company_profiles FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON company_profiles
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE company_profiles;
