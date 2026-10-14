-- +goose Up
-- The shipments report counts a tenant's shipments by label date.
CREATE INDEX shipments_tenant_created_at_idx ON shipments (tenant_id, created_at);

-- +goose Down
DROP INDEX shipments_tenant_created_at_idx;
