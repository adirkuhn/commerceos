-- +goose Up
-- "How many stock units does one purchased item represent" -- almost
-- always 1, set higher when a supplier sells a case/box that isn't itself
-- a catalog product (see domain/procurement.PurchaseOrderLineItem's doc
-- comment, and pms.SKU.UnitsPerCase for the distinct "box IS its own
-- product" case). Learned on the alias too, same as the SKU link itself,
-- so the same supplier text carries its multiplier forward automatically.
ALTER TABLE purchase_order_line_items ADD COLUMN units_per_purchased_item INTEGER NOT NULL DEFAULT 1;
ALTER TABLE supplier_product_aliases ADD COLUMN units_per_purchased_item INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE supplier_product_aliases DROP COLUMN units_per_purchased_item;
ALTER TABLE purchase_order_line_items DROP COLUMN units_per_purchased_item;
