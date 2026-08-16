-- +goose Up

-- skus.barcode had only a non-unique index -- two SKUs sharing a barcode
-- within a tenant let a scan resolve to whichever row Postgres returns
-- first, silently receiving/selling the wrong SKU. Verified no existing
-- duplicates before promoting to UNIQUE.
DROP INDEX IF EXISTS skus_tenant_barcode_idx;
CREATE UNIQUE INDEX skus_tenant_barcode_idx ON skus (tenant_id, barcode) WHERE barcode IS NOT NULL AND barcode <> '';

-- customers.woo_customer_id had no uniqueness guard, unlike every other
-- external-ID mapping in the schema (sales_orders, integration_connections)
-- -- a retried/duplicate Woo customer import could silently duplicate rows.
CREATE UNIQUE INDEX customers_tenant_woo_customer_id_idx ON customers (tenant_id, woo_customer_id) WHERE woo_customer_id IS NOT NULL;

-- warehouses are staff-created (not OCR-extracted like suppliers, where
-- name variance is expected and handled by human review), so a duplicate
-- name within a tenant is just picker/dropdown confusion waiting to happen.
CREATE UNIQUE INDEX warehouses_tenant_id_name_key ON warehouses (tenant_id, name);

-- integration_connections.status had no CHECK, unlike ai_connections.status
-- (same conceptual field) which does.
ALTER TABLE integration_connections ADD CONSTRAINT integration_connections_status_check
    CHECK (status = ANY (ARRAY['connected', 'error']));

-- media used two nullable FKs (product_id/variant_id) instead of a fake
-- polymorphic (type, id) pair -- the right call for real FK integrity, but
-- nothing enforced that exactly one of them is actually set.
ALTER TABLE media ADD CONSTRAINT media_owner_exclusive CHECK (num_nonnulls(product_id, variant_id) = 1);

-- +goose Down

ALTER TABLE media DROP CONSTRAINT media_owner_exclusive;
ALTER TABLE integration_connections DROP CONSTRAINT integration_connections_status_check;
DROP INDEX IF EXISTS warehouses_tenant_id_name_key;
DROP INDEX IF EXISTS customers_tenant_woo_customer_id_idx;
DROP INDEX IF EXISTS skus_tenant_barcode_idx;
CREATE INDEX skus_tenant_barcode_idx ON skus (tenant_id, barcode) WHERE barcode IS NOT NULL AND barcode <> '';
