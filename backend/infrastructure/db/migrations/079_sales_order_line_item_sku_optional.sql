-- +goose Up

-- A WooCommerce order line item can arrive with no matching CommerceOS SKU
-- (a raw SKU string Woo sent that was never imported/linked, or a Woo
-- product with no SKU set at all). Previously ImportWooOrdersUseCase
-- silently dropped the whole line rather than lose data on an unresolved
-- FK -- now it's kept, unlinked, same "not linked to a catalog product
-- yet" pattern purchase_order_line_items.sku_id already uses (see
-- 028_purchase_order_line_items_raw_sku.sql). Name/sku_code/price/tax are
-- already snapshotted directly on the row regardless of sku_id, so an
-- unlinked line still reads correctly.
ALTER TABLE sales_order_line_items ALTER COLUMN sku_id DROP NOT NULL;

-- +goose Down
ALTER TABLE sales_order_line_items ALTER COLUMN sku_id SET NOT NULL;
