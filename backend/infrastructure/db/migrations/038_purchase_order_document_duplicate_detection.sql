-- +goose Up
ALTER TABLE purchase_order_documents ADD COLUMN content_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE purchase_order_documents ADD COLUMN is_potential_duplicate BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX purchase_order_documents_tenant_content_hash_idx ON purchase_order_documents (tenant_id, content_hash);

-- +goose Down
DROP INDEX purchase_order_documents_tenant_content_hash_idx;
ALTER TABLE purchase_order_documents DROP COLUMN is_potential_duplicate;
ALTER TABLE purchase_order_documents DROP COLUMN content_hash;
