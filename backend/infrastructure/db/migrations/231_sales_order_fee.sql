-- +goose Up
-- One extra fee on a sales order (e.g. handling, pallet), entered on the
-- admin's order form beside the shipping cost; counted in total_minor_units
-- like shipping_total_minor_units.
ALTER TABLE sales_orders ADD COLUMN fee_label TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_orders ADD COLUMN fee_total_minor_units BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN IF EXISTS fee_total_minor_units;
ALTER TABLE sales_orders DROP COLUMN IF EXISTS fee_label;
