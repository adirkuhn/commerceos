-- +goose Up
-- B2B customers: a customer is a company (named by its company, KVK
-- number, VAT ID, contact persons) or private (named by the person). Every
-- customer gets a client number, unique per tenant.
ALTER TABLE customers ADD COLUMN customer_type TEXT NOT NULL DEFAULT 'private'
    CONSTRAINT customers_customer_type_check CHECK (customer_type IN ('private', 'company'));
ALTER TABLE customers ADD COLUMN company_name TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN client_number TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN kvk_number TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN vat_number TEXT NOT NULL DEFAULT '';

-- A customer known only by its billing company is a company.
UPDATE customers
SET customer_type = 'company', company_name = trim(billing_company)
WHERE trim(first_name || last_name) = '' AND trim(billing_company) <> '';

-- Number existing customers per tenant, oldest first.
UPDATE customers c
SET client_number = n.rn::TEXT
FROM (SELECT id, row_number() OVER (PARTITION BY tenant_id ORDER BY date_created, id) AS rn FROM customers) n
WHERE n.id = c.id;

CREATE UNIQUE INDEX customers_tenant_client_number_idx ON customers (tenant_id, client_number) WHERE client_number <> '';
CREATE INDEX customers_company_name_trgm_idx ON customers USING gin (immutable_unaccent(company_name) gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS customers_company_name_trgm_idx;
DROP INDEX IF EXISTS customers_tenant_client_number_idx;
ALTER TABLE customers DROP COLUMN IF EXISTS vat_number;
ALTER TABLE customers DROP COLUMN IF EXISTS kvk_number;
ALTER TABLE customers DROP COLUMN IF EXISTS client_number;
ALTER TABLE customers DROP COLUMN IF EXISTS company_name;
ALTER TABLE customers DROP COLUMN IF EXISTS customer_type;
