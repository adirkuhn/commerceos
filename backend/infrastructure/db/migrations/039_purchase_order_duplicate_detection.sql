-- +goose Up
ALTER TABLE purchase_orders ADD COLUMN is_potential_duplicate BOOLEAN NOT NULL DEFAULT false;
-- SET NULL (not CASCADE): if the PO this one was flagged as a duplicate of
-- gets deleted, this PO must survive -- only the pointer goes stale, so it
-- clears rather than taking this row down with it.
ALTER TABLE purchase_orders ADD COLUMN duplicate_of_purchase_order_id UUID REFERENCES purchase_orders(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE purchase_orders DROP COLUMN duplicate_of_purchase_order_id;
ALTER TABLE purchase_orders DROP COLUMN is_potential_duplicate;
