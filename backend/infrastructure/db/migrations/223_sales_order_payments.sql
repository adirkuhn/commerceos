-- +goose Up
-- Every part that paid a sales order (domain/oms.SalesOrderPayment) -- a
-- POS sale can be split over several payments with any mix of methods
-- (card+card, cash+card, ...). position keeps the order the parts were
-- taken in. An order with no rows reported no breakdown; its
-- payment_method/transaction_id are then the only payment record.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE sales_order_payments (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL,
    sales_order_id     UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    position           INTEGER     NOT NULL,
    method             TEXT        NOT NULL,
    amount_minor_units BIGINT      NOT NULL CHECK (amount_minor_units > 0),
    transaction_id     TEXT        NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sales_order_payments_sales_order_id_idx ON sales_order_payments (sales_order_id);

ALTER TABLE sales_order_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_payments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_order_payments
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS sales_order_payments;
