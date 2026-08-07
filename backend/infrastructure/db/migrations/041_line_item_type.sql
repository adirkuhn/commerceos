-- +goose Up
-- Classifies a line item as a product or a non-product charge (shipping,
-- discount, fee, credit note, other charge) -- charge rows must never be
-- forced through SKU matching. Empty string means "product" (see
-- domain/procurement.LineItemType.IsProduct) -- the deterministic CSV/XLSX
-- parser has no classification signal and always leaves this unset.
ALTER TABLE purchase_order_line_items ADD COLUMN line_type TEXT NOT NULL DEFAULT '';
ALTER TABLE purchase_order_line_items ADD COLUMN line_type_confirmed BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE purchase_order_line_items DROP COLUMN line_type_confirmed;
ALTER TABLE purchase_order_line_items DROP COLUMN line_type;
