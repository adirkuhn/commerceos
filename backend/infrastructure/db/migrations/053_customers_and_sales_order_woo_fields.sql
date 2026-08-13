-- +goose Up
-- customers mirrors WooCommerce's customer object field-for-field (see
-- domain/customers.Customer's doc comment) so a customer synced from (or
-- later pushed to) Woo round-trips cleanly.
CREATE TABLE customers (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL,
    woo_customer_id    BIGINT,
    email              TEXT        NOT NULL DEFAULT '',
    first_name         TEXT        NOT NULL DEFAULT '',
    last_name          TEXT        NOT NULL DEFAULT '',
    username           TEXT        NOT NULL DEFAULT '',
    role               TEXT        NOT NULL DEFAULT '',
    billing_first_name TEXT        NOT NULL DEFAULT '',
    billing_last_name  TEXT        NOT NULL DEFAULT '',
    billing_company    TEXT        NOT NULL DEFAULT '',
    billing_address_1  TEXT        NOT NULL DEFAULT '',
    billing_address_2  TEXT        NOT NULL DEFAULT '',
    billing_city       TEXT        NOT NULL DEFAULT '',
    billing_state      TEXT        NOT NULL DEFAULT '',
    billing_postcode   TEXT        NOT NULL DEFAULT '',
    billing_country    TEXT        NOT NULL DEFAULT '',
    billing_email      TEXT        NOT NULL DEFAULT '',
    billing_phone      TEXT        NOT NULL DEFAULT '',
    shipping_first_name TEXT       NOT NULL DEFAULT '',
    shipping_last_name  TEXT       NOT NULL DEFAULT '',
    shipping_company    TEXT       NOT NULL DEFAULT '',
    shipping_address_1  TEXT       NOT NULL DEFAULT '',
    shipping_address_2  TEXT       NOT NULL DEFAULT '',
    shipping_city       TEXT       NOT NULL DEFAULT '',
    shipping_state      TEXT       NOT NULL DEFAULT '',
    shipping_postcode   TEXT       NOT NULL DEFAULT '',
    shipping_country    TEXT       NOT NULL DEFAULT '',
    shipping_phone      TEXT       NOT NULL DEFAULT '',
    is_paying_customer BOOLEAN     NOT NULL DEFAULT FALSE,
    avatar_url         TEXT        NOT NULL DEFAULT '',
    date_created       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    date_modified      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX customers_tenant_id_idx ON customers (tenant_id);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON customers
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- sales_orders/sales_order_line_items gain the rest of WooCommerce's order
-- object fields (see domain/oms.SalesOrder's doc comment) -- 052 only had
-- the minimal fields ReceivePOSSalesOrderUseCase needed at the time.
ALTER TABLE sales_orders ADD COLUMN woo_order_id BIGINT;
ALTER TABLE sales_orders ADD COLUMN number TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN order_key TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN status TEXT NOT NULL DEFAULT 'completed';
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check
    CHECK (status IN ('pending', 'processing', 'on-hold', 'completed', 'cancelled', 'refunded', 'failed'));
ALTER TABLE sales_orders ADD COLUMN customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;
ALTER TABLE sales_orders ADD COLUMN billing_first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_last_name TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_company TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_address_1 TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_address_2 TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_city TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_state TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_postcode TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_country TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_email TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN billing_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_last_name TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_company TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_address_1 TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_address_2 TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_city TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_state TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_postcode TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_country TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN shipping_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN customer_note TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN discount_total_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_orders ADD COLUMN shipping_total_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_orders ADD COLUMN tax_total_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_orders ADD COLUMN payment_method TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN payment_method_title TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN transaction_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN date_paid TIMESTAMPTZ;
ALTER TABLE sales_orders ADD COLUMN date_completed TIMESTAMPTZ;
ALTER TABLE sales_orders ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX sales_orders_customer_id_idx ON sales_orders (customer_id);

-- line_items.unit_price_minor_units is replaced by the price/subtotal/total
-- split Woo actually uses -- safe to drop outright, this table has no
-- production data yet (added this session).
ALTER TABLE sales_order_line_items DROP COLUMN unit_price_minor_units;
ALTER TABLE sales_order_line_items ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_order_line_items ADD COLUMN sku_code TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_order_line_items ADD COLUMN price_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_order_line_items ADD COLUMN subtotal_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_order_line_items ADD COLUMN subtotal_tax_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_order_line_items ADD COLUMN total_minor_units BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales_order_line_items ADD COLUMN total_tax_minor_units BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sales_order_line_items DROP COLUMN total_tax_minor_units;
ALTER TABLE sales_order_line_items DROP COLUMN total_minor_units;
ALTER TABLE sales_order_line_items DROP COLUMN subtotal_tax_minor_units;
ALTER TABLE sales_order_line_items DROP COLUMN subtotal_minor_units;
ALTER TABLE sales_order_line_items DROP COLUMN price_minor_units;
ALTER TABLE sales_order_line_items DROP COLUMN sku_code;
ALTER TABLE sales_order_line_items DROP COLUMN name;
ALTER TABLE sales_order_line_items ADD COLUMN unit_price_minor_units BIGINT NOT NULL DEFAULT 0;

DROP INDEX sales_orders_customer_id_idx;
ALTER TABLE sales_orders DROP COLUMN updated_at;
ALTER TABLE sales_orders DROP COLUMN date_completed;
ALTER TABLE sales_orders DROP COLUMN date_paid;
ALTER TABLE sales_orders DROP COLUMN transaction_id;
ALTER TABLE sales_orders DROP COLUMN payment_method_title;
ALTER TABLE sales_orders DROP COLUMN payment_method;
ALTER TABLE sales_orders DROP COLUMN tax_total_minor_units;
ALTER TABLE sales_orders DROP COLUMN shipping_total_minor_units;
ALTER TABLE sales_orders DROP COLUMN discount_total_minor_units;
ALTER TABLE sales_orders DROP COLUMN customer_note;
ALTER TABLE sales_orders DROP COLUMN shipping_phone;
ALTER TABLE sales_orders DROP COLUMN shipping_country;
ALTER TABLE sales_orders DROP COLUMN shipping_postcode;
ALTER TABLE sales_orders DROP COLUMN shipping_state;
ALTER TABLE sales_orders DROP COLUMN shipping_city;
ALTER TABLE sales_orders DROP COLUMN shipping_address_2;
ALTER TABLE sales_orders DROP COLUMN shipping_address_1;
ALTER TABLE sales_orders DROP COLUMN shipping_company;
ALTER TABLE sales_orders DROP COLUMN shipping_last_name;
ALTER TABLE sales_orders DROP COLUMN shipping_first_name;
ALTER TABLE sales_orders DROP COLUMN billing_phone;
ALTER TABLE sales_orders DROP COLUMN billing_email;
ALTER TABLE sales_orders DROP COLUMN billing_country;
ALTER TABLE sales_orders DROP COLUMN billing_postcode;
ALTER TABLE sales_orders DROP COLUMN billing_state;
ALTER TABLE sales_orders DROP COLUMN billing_city;
ALTER TABLE sales_orders DROP COLUMN billing_address_2;
ALTER TABLE sales_orders DROP COLUMN billing_address_1;
ALTER TABLE sales_orders DROP COLUMN billing_company;
ALTER TABLE sales_orders DROP COLUMN billing_last_name;
ALTER TABLE sales_orders DROP COLUMN billing_first_name;
ALTER TABLE sales_orders DROP COLUMN customer_id;
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
ALTER TABLE sales_orders DROP COLUMN status;
ALTER TABLE sales_orders DROP COLUMN order_key;
ALTER TABLE sales_orders DROP COLUMN number;
ALTER TABLE sales_orders DROP COLUMN woo_order_id;

DROP TABLE customers;
