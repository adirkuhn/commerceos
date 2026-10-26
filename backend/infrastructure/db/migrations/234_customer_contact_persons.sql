-- +goose Up
-- People to deal with at a company customer. Removed with the customer.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE customer_contact_persons (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    customer_id UUID        NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    first_name  TEXT        NOT NULL DEFAULT '',
    last_name   TEXT        NOT NULL DEFAULT '',
    role        TEXT        NOT NULL DEFAULT '',
    email       TEXT        NOT NULL DEFAULT '',
    phone       TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT customer_contact_persons_name_check CHECK (trim(first_name || last_name) <> '')
);

CREATE INDEX customer_contact_persons_customer_idx ON customer_contact_persons (customer_id, created_at);
CREATE INDEX customer_contact_persons_name_trgm_idx ON customer_contact_persons
    USING gin (immutable_unaccent(first_name || ' ' || last_name) gin_trgm_ops);

ALTER TABLE customer_contact_persons ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_contact_persons FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON customer_contact_persons
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS customer_contact_persons;
