-- +goose Up

-- domain/oms.SalesOrderStatus already defines SalesOrderStatusShipped, used
-- by application/shipping's order_status_change webhook handler, but the
-- DB CHECK constraint added in 053 never learned about it -- any write of
-- status='shipped' has been one CHECK-violation away from failing outright.
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check
    CHECK (status IN ('pending', 'processing', 'on-hold', 'shipped', 'completed', 'cancelled', 'refunded', 'failed'));

-- +goose Down
ALTER TABLE sales_orders DROP CONSTRAINT sales_orders_status_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check
    CHECK (status IN ('pending', 'processing', 'on-hold', 'completed', 'cancelled', 'refunded', 'failed'));
