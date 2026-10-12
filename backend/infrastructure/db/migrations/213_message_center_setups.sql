-- +goose Up
-- One row per tenant that was given the Message Center's default templates
-- and rules (application/messagecenter.EnsureMessageCenterDefaultsUseCase),
-- so defaults the tenant deletes later are not created again.
CREATE TABLE message_center_setups (
    tenant_id  UUID        PRIMARY KEY,
    seeded_at  TIMESTAMPTZ NOT NULL
);

ALTER TABLE message_center_setups ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_center_setups FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON message_center_setups
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS message_center_setups;
