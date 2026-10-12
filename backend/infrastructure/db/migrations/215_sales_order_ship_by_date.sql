-- +goose Up

-- The calendar date an order must be handed to the carrier by (see
-- domain/oms.SalesOrder.ShipByDate). Only Bol imports set it today, derived
-- from Bol's fulfilment delivery dates; NULL for every other order and for
-- Bol orders imported before this existed. Powers the admin "Today" page's
-- "orders to ship today" count. sales_orders is already tenant-scoped with
-- RLS, so no new policy is needed.
ALTER TABLE sales_orders ADD COLUMN ship_by_date DATE;

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN ship_by_date;
