-- +goose Up
-- Append-only log of every WhatsApp message (inbound and outbound reply)
-- the integration has seen -- distinct from whatsapp_conversations (137),
-- which holds only the CURRENT checkout state and is overwritten every
-- turn. customer_id is nullable and set only when the phone number
-- matched an existing customer at the time the message was recorded (see
-- domain/customers.CustomerRepository.FindByPhoneE164) -- a message from
-- an unmatched number is still logged, just not linked to anyone. No FK
-- to customers: same "no local FK, tenant-scoped id string" convention as
-- sales_order_activities' user_id.
CREATE TABLE whatsapp_messages (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    phone_e164  TEXT        NOT NULL,
    customer_id UUID        NULL,
    direction   TEXT        NOT NULL,
    body        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT whatsapp_messages_direction_check CHECK (direction IN ('inbound', 'outbound'))
);

CREATE INDEX whatsapp_messages_tenant_id_idx ON whatsapp_messages (tenant_id);
CREATE INDEX whatsapp_messages_phone_idx ON whatsapp_messages (tenant_id, phone_e164, created_at DESC);
CREATE INDEX whatsapp_messages_customer_idx ON whatsapp_messages (tenant_id, customer_id, created_at DESC) WHERE customer_id IS NOT NULL;

ALTER TABLE whatsapp_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE whatsapp_messages FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON whatsapp_messages
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE whatsapp_messages;
