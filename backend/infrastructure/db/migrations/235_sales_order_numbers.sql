-- +goose Up
-- One row per tenant: the last order number handed out to an order made in
-- CommerceOS itself (admin form, quote, WhatsApp, storefront), shown as
-- CO-<n>. Woo, Bol and POS orders keep the number their channel gave them.
-- Numbers are never reused, even after the newest order is deleted.
CREATE TABLE sales_order_number_counters (
    tenant_id   UUID   PRIMARY KEY,
    last_number BIGINT NOT NULL
);

ALTER TABLE sales_order_number_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_number_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sales_order_number_counters
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Orders made here so far had no number (the page showed their UUID):
-- number them per tenant, oldest first, from CO-1001.
WITH numbered AS (
    SELECT id, tenant_id, 1000 + row_number() OVER (PARTITION BY tenant_id ORDER BY created_at, id) AS n
    FROM sales_orders
    WHERE number = '' AND source IN ('manual', 'whatsapp', 'internal')
)
UPDATE sales_orders o SET number = 'CO-' || numbered.n
FROM numbered WHERE o.id = numbered.id;

INSERT INTO sales_order_number_counters (tenant_id, last_number)
SELECT tenant_id, max(substring(number FROM 4)::bigint) FROM sales_orders
WHERE number ~ '^CO-[0-9]+$'
GROUP BY tenant_id;

-- +goose Down
DROP TABLE IF EXISTS sales_order_number_counters;
