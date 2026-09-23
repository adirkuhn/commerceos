-- +goose Up

-- Per-tenant notification delivery choice: for each category (imports,
-- purchase documents, ...) whether it shows in the admin bell and/or goes
-- by email, plus who receives the emails. See
-- domain/notifications/preferences.go. No row = defaults (bell on, email off).
CREATE TABLE notification_preferences (
    tenant_id        UUID        PRIMARY KEY,
    categories       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    email_recipients JSONB       NOT NULL DEFAULT '[]'::jsonb,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON notification_preferences
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE notification_preferences;
