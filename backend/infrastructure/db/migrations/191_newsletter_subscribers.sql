-- +goose Up
-- Store mailing list with double opt-in (domain/newsletter). One row per
-- address per tenant; deleting a row is the right-to-erasure path.
CREATE TABLE newsletter_subscribers (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    sales_channel_id     UUID        REFERENCES sales_channels(id) ON DELETE SET NULL,
    email                TEXT        NOT NULL,
    status               TEXT        NOT NULL CHECK (status IN ('pending', 'confirmed', 'unsubscribed')),
    confirm_token_hash   TEXT,
    unsubscribe_token    TEXT        NOT NULL,
    consent_at           TIMESTAMPTZ NOT NULL,
    confirmation_sent_at TIMESTAMPTZ NOT NULL,
    confirmed_at         TIMESTAMPTZ,
    unsubscribed_at      TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT newsletter_subscribers_tenant_email_unique UNIQUE (tenant_id, email),
    CONSTRAINT newsletter_subscribers_unsubscribe_token_unique UNIQUE (unsubscribe_token)
);
CREATE UNIQUE INDEX newsletter_subscribers_confirm_token ON newsletter_subscribers (confirm_token_hash) WHERE confirm_token_hash IS NOT NULL;
CREATE INDEX newsletter_subscribers_tenant_created_idx ON newsletter_subscribers (tenant_id, created_at DESC);

ALTER TABLE newsletter_subscribers ENABLE ROW LEVEL SECURITY;
ALTER TABLE newsletter_subscribers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON newsletter_subscribers
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE newsletter_subscribers;
