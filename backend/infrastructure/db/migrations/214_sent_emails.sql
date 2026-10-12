-- +goose Up
-- The tenant's email history (domain/messagecenter.SentEmail): every email
-- CommerceOS sent, failed to send or -- for automation rules -- skipped,
-- with the full message. Kept until the tenant deletes entries.
CREATE TABLE sent_emails (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    source          TEXT        NOT NULL CHECK (source IN ('automation', 'notification', 'sign_in_link', 'newsletter')),
    recipient       TEXT        NOT NULL DEFAULT '',
    subject         TEXT        NOT NULL DEFAULT '',
    html_body       TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL CHECK (status IN ('sent', 'failed', 'skipped')),
    reason          TEXT        NOT NULL DEFAULT '',
    rule_id         UUID,
    template_id     UUID,
    trigger_subject TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX sent_emails_tenant_created_idx ON sent_emails (tenant_id, created_at DESC);

ALTER TABLE sent_emails ENABLE ROW LEVEL SECURITY;
ALTER TABLE sent_emails FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sent_emails
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS sent_emails;
