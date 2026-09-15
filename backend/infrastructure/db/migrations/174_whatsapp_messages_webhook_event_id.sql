-- +goose Up
-- HandleInboundMessageUseCase.Execute re-runs from scratch on every NATS
-- redelivery (JetStream's own retry, triggered whenever the reply send
-- fails -- see infrastructure/consumers.WebhookConsumer: the outer
-- processed_events dedup only marks an event processed on full success,
-- so a redelivery legitimately retries the whole handler). Without this,
-- the unconditional inbound Create ran again on every retry, logging the
-- same customer message N times. webhook_event_id lets Create skip a
-- duplicate insert for the same (tenant, event) instead -- nullable
-- because it only applies to inbound messages recorded from a webhook
-- envelope, never outbound replies or any other message this table might
-- one day carry.
-- A plain (non-partial) unique index: Postgres never treats two NULLs as
-- conflicting in a unique index, so outbound/other rows that never set
-- webhook_event_id insert freely -- only two rows sharing the same
-- (tenant_id, webhook_event_id) actually conflict, letting Create's
-- ON CONFLICT DO NOTHING (see WhatsAppMessageRepo.Create) apply
-- unconditionally with no partial-index predicate to match.
ALTER TABLE whatsapp_messages ADD COLUMN webhook_event_id TEXT NULL;

CREATE UNIQUE INDEX whatsapp_messages_tenant_webhook_event_id_idx
    ON whatsapp_messages (tenant_id, webhook_event_id);

-- +goose Down
DROP INDEX whatsapp_messages_tenant_webhook_event_id_idx;
ALTER TABLE whatsapp_messages DROP COLUMN webhook_event_id;
