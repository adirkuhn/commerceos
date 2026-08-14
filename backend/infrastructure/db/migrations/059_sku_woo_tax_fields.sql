-- +goose Up
-- Informational provenance only -- never used by tax calculation (that
-- always goes through tax_class_id -> tax_classes/tax_rates). Populated
-- only when a SKU is imported from WooCommerce, displayed on the
-- product's WooCommerce tab so staff can see what Woo actually sent
-- ("" tax_class means Woo's own Standard rate).
ALTER TABLE skus ADD COLUMN woo_tax_class_slug TEXT;
ALTER TABLE skus ADD COLUMN woo_tax_status TEXT;

-- +goose Down
ALTER TABLE skus DROP COLUMN woo_tax_status;
ALTER TABLE skus DROP COLUMN woo_tax_class_slug;
