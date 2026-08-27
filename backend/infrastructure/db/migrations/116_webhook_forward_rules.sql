-- +goose Up

-- Tenant-configured fan-out: "when a webhook of this provider/topic lands,
-- also POST it to these URLs" -- the Settings/Integrations UI feature. One
-- row per (provider, topic, target) so a topic can fan out to several URLs
-- (multiple rows) the way a rule engine's routing table would. topic = ''
-- matches every topic for that provider (needed for providers, like
-- MyParcel today, that don't send a topic per delivery -- see
-- ProcessWebhookEventUseCase's topicHeader comment).
CREATE TABLE webhook_forward_rules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    provider    TEXT        NOT NULL,
    topic       TEXT        NOT NULL DEFAULT '',
    target_url  TEXT        NOT NULL,
    active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX webhook_forward_rules_tenant_id_idx ON webhook_forward_rules (tenant_id);
CREATE INDEX webhook_forward_rules_lookup_idx ON webhook_forward_rules (tenant_id, provider, topic) WHERE active;

ALTER TABLE webhook_forward_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_forward_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON webhook_forward_rules
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- One row per (matched rule, received webhook) -- the delivery queue
-- cmd/webhook-forwarder drains. Inserted by ProcessWebhookEventUseCase
-- INSIDE the same transaction that saves the webhook_events row (same
-- durability guarantee as the domain-event outbox: the intent to forward
-- is never lost even if the HTTP delivery itself later fails). Unlike
-- outbox_messages (which has no tenant_id/RLS at all -- pure system
-- plumbing with no per-tenant UI), this table backs a tenant-visible
-- delivery log, so it stays RLS-scoped like every other tenant table; the
-- worker lists across tenants under tenant.WithSuperAdmin, same convention
-- as SyncScheduleRepository.ListDue.
CREATE TABLE webhook_forward_deliveries (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    rule_id          UUID        NOT NULL REFERENCES webhook_forward_rules(id) ON DELETE CASCADE,
    webhook_event_id UUID        NOT NULL,
    provider         TEXT        NOT NULL,
    topic            TEXT        NOT NULL DEFAULT '',
    target_url       TEXT        NOT NULL,
    payload          JSONB       NOT NULL,
    status           TEXT        NOT NULL DEFAULT 'pending',
    attempts         INT         NOT NULL DEFAULT 0,
    last_error       TEXT        NOT NULL DEFAULT '',
    next_attempt_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT webhook_forward_deliveries_status_check CHECK (status IN ('pending', 'delivered', 'failed')),
    -- Redelivery of the same JetStream envelope must not fan out twice --
    -- see ProcessWebhookEventUseCase's doc comment on why this constraint
    -- (not just app-level care) is what makes the insert idempotent.
    CONSTRAINT webhook_forward_deliveries_event_rule_unique UNIQUE (webhook_event_id, rule_id)
);

CREATE INDEX webhook_forward_deliveries_tenant_id_idx ON webhook_forward_deliveries (tenant_id);
CREATE INDEX webhook_forward_deliveries_pending_idx ON webhook_forward_deliveries (next_attempt_at) WHERE status = 'pending';

ALTER TABLE webhook_forward_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_forward_deliveries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON webhook_forward_deliveries
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE webhook_forward_deliveries;
DROP TABLE webhook_forward_rules;
