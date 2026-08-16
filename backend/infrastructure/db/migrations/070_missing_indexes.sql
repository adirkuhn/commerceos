-- +goose Up

-- stock_movements is the highest-write-volume table in the schema (every
-- receipt/sale/adjustment/transfer/write-off inserts a row) and had no
-- index on tenant_id or warehouse_id -- every RLS-filtered query and every
-- "current stock in this warehouse" read was falling back to a seq scan or
-- the much-less-selective sku_id/batch_id indexes.
CREATE INDEX stock_movements_tenant_id_idx ON stock_movements (tenant_id);
CREATE INDEX stock_movements_warehouse_id_idx ON stock_movements (warehouse_id);

CREATE INDEX batches_tenant_id_idx ON batches (tenant_id);
CREATE INDEX batches_warehouse_id_idx ON batches (warehouse_id);

-- skus.variant_id is a mandatory FK loaded on essentially every PMS product
-- read (loading a product's SKUs from its variants), unindexed until now.
CREATE INDEX skus_variant_id_idx ON skus (variant_id);

CREATE INDEX import_jobs_tenant_id_idx ON import_jobs (tenant_id);
CREATE INDEX product_channel_listings_tenant_id_idx ON product_channel_listings (tenant_id);
CREATE INDEX product_channel_listings_connection_id_idx ON product_channel_listings (connection_id);
CREATE INDEX purchase_order_line_items_tenant_id_idx ON purchase_order_line_items (tenant_id);
CREATE INDEX purchase_order_line_items_sku_id_idx ON purchase_order_line_items (sku_id);
CREATE INDEX sales_order_line_items_tenant_id_idx ON sales_order_line_items (tenant_id);
CREATE INDEX sales_order_line_items_sku_id_idx ON sales_order_line_items (sku_id);
CREATE INDEX purchase_orders_warehouse_id_idx ON purchase_orders (warehouse_id);
CREATE INDEX sales_orders_warehouse_id_idx ON sales_orders (warehouse_id);
CREATE INDEX sales_channel_product_overrides_product_id_idx ON sales_channel_product_overrides (product_id);
CREATE INDEX supplier_product_aliases_sku_id_idx ON supplier_product_aliases (sku_id);

-- +goose Down

DROP INDEX IF EXISTS supplier_product_aliases_sku_id_idx;
DROP INDEX IF EXISTS sales_channel_product_overrides_product_id_idx;
DROP INDEX IF EXISTS sales_orders_warehouse_id_idx;
DROP INDEX IF EXISTS purchase_orders_warehouse_id_idx;
DROP INDEX IF EXISTS sales_order_line_items_sku_id_idx;
DROP INDEX IF EXISTS sales_order_line_items_tenant_id_idx;
DROP INDEX IF EXISTS purchase_order_line_items_sku_id_idx;
DROP INDEX IF EXISTS purchase_order_line_items_tenant_id_idx;
DROP INDEX IF EXISTS product_channel_listings_connection_id_idx;
DROP INDEX IF EXISTS product_channel_listings_tenant_id_idx;
DROP INDEX IF EXISTS import_jobs_tenant_id_idx;
DROP INDEX IF EXISTS skus_variant_id_idx;
DROP INDEX IF EXISTS batches_warehouse_id_idx;
DROP INDEX IF EXISTS batches_tenant_id_idx;
DROP INDEX IF EXISTS stock_movements_warehouse_id_idx;
DROP INDEX IF EXISTS stock_movements_tenant_id_idx;
