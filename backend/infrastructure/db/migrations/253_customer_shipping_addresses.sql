-- +goose Up
-- A customer's shipping addresses: any number, exactly one default while
-- any exist. customers.shipping_* stays a copy of the address in use (the
-- default, or billing when ship_to_billing) for everything that reads one
-- address. Removed with the customer. commerceos_app gets its grants from
-- 020's ALTER DEFAULT PRIVILEGES.
ALTER TABLE customers ADD COLUMN ship_to_billing BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE customer_shipping_addresses (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    customer_id UUID        NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    label       TEXT        NOT NULL DEFAULT '',
    first_name  TEXT        NOT NULL DEFAULT '',
    last_name   TEXT        NOT NULL DEFAULT '',
    company     TEXT        NOT NULL DEFAULT '',
    address_1   TEXT        NOT NULL DEFAULT '',
    address_2   TEXT        NOT NULL DEFAULT '',
    city        TEXT        NOT NULL DEFAULT '',
    state       TEXT        NOT NULL DEFAULT '',
    postcode    TEXT        NOT NULL DEFAULT '',
    country     TEXT        NOT NULL DEFAULT '',
    phone       TEXT        NOT NULL DEFAULT '',
    is_default  BOOLEAN     NOT NULL DEFAULT FALSE,
    position    INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX customer_shipping_addresses_customer_idx ON customer_shipping_addresses (customer_id, position);
CREATE UNIQUE INDEX customer_shipping_addresses_one_default_idx ON customer_shipping_addresses (customer_id) WHERE is_default;

ALTER TABLE customer_shipping_addresses ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_shipping_addresses FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON customer_shipping_addresses
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Every customer's current shipping address becomes their default.
INSERT INTO customer_shipping_addresses (tenant_id, customer_id, first_name, last_name, company, address_1, address_2,
                                         city, state, postcode, country, phone, is_default, position, created_at, updated_at)
SELECT tenant_id, id, shipping_first_name, shipping_last_name, shipping_company, shipping_address_1, shipping_address_2,
       shipping_city, shipping_state, shipping_postcode, shipping_country, shipping_phone, TRUE, 0, now(), now()
FROM customers
WHERE trim(shipping_first_name || shipping_last_name || shipping_company || shipping_address_1 || shipping_address_2
           || shipping_city || shipping_state || shipping_postcode || shipping_country || shipping_phone) <> '';

-- +goose Down
DROP TABLE IF EXISTS customer_shipping_addresses;
ALTER TABLE customers DROP COLUMN IF EXISTS ship_to_billing;
