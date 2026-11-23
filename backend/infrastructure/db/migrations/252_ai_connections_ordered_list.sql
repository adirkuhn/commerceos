-- +goose Up
-- A tenant keeps an ordered list of AI connections instead of one: every AI
-- feature tries them in position order and moves to the next on any
-- failure. The same provider may appear twice (two free-tier keys).
-- Existing connections become the first (position 0), enabled.
ALTER TABLE ai_connections DROP CONSTRAINT ai_connections_tenant_unique;
ALTER TABLE ai_connections ADD COLUMN position INT NOT NULL DEFAULT 0;
ALTER TABLE ai_connections ADD COLUMN label TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_connections ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX ai_connections_tenant_position_idx ON ai_connections (tenant_id, position);

-- +goose Down
-- Keeps only each tenant's first connection: the old schema allows one.
DELETE FROM ai_connections c USING ai_connections keep
WHERE c.tenant_id = keep.tenant_id
  AND (keep.position, keep.created_at, keep.id) < (c.position, c.created_at, c.id);
DROP INDEX ai_connections_tenant_position_idx;
ALTER TABLE ai_connections DROP COLUMN enabled;
ALTER TABLE ai_connections DROP COLUMN label;
ALTER TABLE ai_connections DROP COLUMN position;
ALTER TABLE ai_connections ADD CONSTRAINT ai_connections_tenant_unique UNIQUE (tenant_id);
