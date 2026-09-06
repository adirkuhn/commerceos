-- +goose Up
-- One row per (tenant_id, phone_e164) -- the in-progress WhatsApp checkout
-- state machine (domain/whatsapp.Conversation). Upserted on every inbound
-- message; State/Cart/PendingCustomer* reset once an order is placed or
-- the flow is cancelled, not deleted (row persists so the phone number's
-- next conversation just starts from Idle).
CREATE TABLE whatsapp_conversations (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    phone_e164                TEXT        NOT NULL,
    state                     TEXT        NOT NULL DEFAULT '',
    cart                      JSONB       NOT NULL DEFAULT '[]',
    pending_customer_name     TEXT        NOT NULL DEFAULT '',
    pending_customer_email    TEXT        NOT NULL DEFAULT '',
    pending_customer_address  TEXT        NOT NULL DEFAULT '',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT whatsapp_conversations_tenant_phone_unique UNIQUE (tenant_id, phone_e164)
);

CREATE INDEX whatsapp_conversations_tenant_id_idx ON whatsapp_conversations (tenant_id);

ALTER TABLE whatsapp_conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE whatsapp_conversations FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON whatsapp_conversations
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE whatsapp_conversations;
