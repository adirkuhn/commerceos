-- +goose Up

-- "Was this message processed" -- ProcessWebhookEventUseCase sets this
-- right after invoking a per-topic dispatch handler (myparcel today,
-- see application/webhooks.ProcessWebhookEventUseCase). Every other
-- provider/topic (no dispatch handler exists) stays 'received' forever --
-- its normal, successful terminal state, not an unprocessed one.
ALTER TABLE webhook_events ADD COLUMN status TEXT NOT NULL DEFAULT 'received';
ALTER TABLE webhook_events ADD CONSTRAINT webhook_events_status_check
    CHECK (status IN ('received', 'processed', 'failed'));
ALTER TABLE webhook_events ADD COLUMN dispatch_error TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE webhook_events DROP COLUMN dispatch_error;
ALTER TABLE webhook_events DROP CONSTRAINT webhook_events_status_check;
ALTER TABLE webhook_events DROP COLUMN status;
