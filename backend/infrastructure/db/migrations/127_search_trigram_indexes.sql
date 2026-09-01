-- +goose Up
-- Global search (Cmd+K) extends the same accent-insensitive + typo-tolerant
-- matching Products already has (022_unaccent_extension.sql,
-- 077_pg_trgm_extension.sql's immutable_unaccent()) to Suppliers, Customers,
-- and Sales Orders. Email columns are deliberately NOT trigram-indexed here
-- -- fuzzy-matching an email address (e.g. "jane@example.com" typo-tolerant
-- against "jane@exampel.com") produces confusing near-miss results for what
-- is really an exact-token lookup; those stay plain ILIKE, unindexed (small
-- per-tenant row counts, and Postgres can still use the existing btree for
-- an exact/prefix hit).
CREATE INDEX idx_suppliers_name_trgm ON suppliers USING gin (immutable_unaccent(name) gin_trgm_ops);

CREATE INDEX idx_customers_full_name_trgm ON customers USING gin (immutable_unaccent(first_name || ' ' || last_name) gin_trgm_ops);

CREATE INDEX idx_sales_orders_number_trgm ON sales_orders USING gin (immutable_unaccent(number) gin_trgm_ops);
CREATE INDEX idx_sales_orders_billing_name_trgm ON sales_orders USING gin (immutable_unaccent(billing_first_name || ' ' || billing_last_name) gin_trgm_ops);

-- +goose Down
DROP INDEX idx_sales_orders_billing_name_trgm;
DROP INDEX idx_sales_orders_number_trgm;
DROP INDEX idx_customers_full_name_trgm;
DROP INDEX idx_suppliers_name_trgm;
