-- +goose Up
CREATE TABLE automation_rules (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    name             TEXT        NOT NULL,
    trigger_subject  TEXT        NOT NULL,
    conditions       JSONB       NOT NULL DEFAULT '[]',
    template_id      UUID        NOT NULL REFERENCES message_templates(id) ON DELETE CASCADE,
    channel_override TEXT,
    enabled          BOOLEAN     NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX automation_rules_tenant_id_idx ON automation_rules (tenant_id);
CREATE INDEX automation_rules_trigger_subject_idx ON automation_rules (trigger_subject);
CREATE INDEX automation_rules_template_id_idx ON automation_rules (template_id);

ALTER TABLE automation_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE automation_rules FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON automation_rules
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE automation_rules;
