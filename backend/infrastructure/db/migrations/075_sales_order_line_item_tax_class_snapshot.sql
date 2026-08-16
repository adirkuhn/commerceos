-- +goose Up

-- sales_order_line_items already snapshots name/sku_code/barcode/price for
-- historical fidelity (a line must read correctly even if the product is
-- later renamed or its price changes), but not which tax classification
-- applied -- tax_rates can change over time, so a line's tax basis can't
-- currently be explained/audited after the fact. Nullable + ON DELETE SET
-- NULL: this is a best-effort snapshot of the SKU's tax_class_id at sale
-- time, not a hard requirement (older rows and any SKU without a tax
-- class have it NULL).
ALTER TABLE sales_order_line_items ADD COLUMN tax_class_id UUID REFERENCES tax_classes(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE sales_order_line_items DROP COLUMN tax_class_id;
