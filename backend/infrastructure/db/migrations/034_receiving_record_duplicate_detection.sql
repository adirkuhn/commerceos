-- +goose Up
ALTER TABLE receiving_records ADD COLUMN content_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE receiving_records ADD COLUMN is_potential_duplicate BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX receiving_records_tenant_content_hash_idx ON receiving_records (tenant_id, content_hash);

-- +goose Down
DROP INDEX receiving_records_tenant_content_hash_idx;
ALTER TABLE receiving_records DROP COLUMN is_potential_duplicate;
ALTER TABLE receiving_records DROP COLUMN content_hash;
