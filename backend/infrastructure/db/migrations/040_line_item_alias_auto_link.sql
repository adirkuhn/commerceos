-- +goose Up
-- Marks a line item's SKUID as having come from a learned
-- SupplierProductAlias guess rather than an exact-SKU match or an explicit
-- human link -- lets the app safely clear + redo it if the PO's supplier
-- changes, without ever touching a deliberately-set link.
ALTER TABLE purchase_order_line_items ADD COLUMN sku_auto_linked_from_alias BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE purchase_order_line_items DROP COLUMN sku_auto_linked_from_alias;
