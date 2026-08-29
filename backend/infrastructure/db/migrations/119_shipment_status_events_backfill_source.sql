-- +goose Up

-- BackfillShipmentWebhookDispatchUseCase (a one-time reprocessing of
-- webhooks received before dispatch-status tracking existed) writes a
-- 'backfill' entry to the same history table webhook/reconciliation/manual
-- already write to -- widen the source check constraint to allow it.
ALTER TABLE shipment_status_events DROP CONSTRAINT shipment_status_events_source_check;
ALTER TABLE shipment_status_events ADD CONSTRAINT shipment_status_events_source_check
    CHECK (source IN ('webhook', 'reconciliation', 'manual', 'backfill'));

-- +goose Down
ALTER TABLE shipment_status_events DROP CONSTRAINT shipment_status_events_source_check;
ALTER TABLE shipment_status_events ADD CONSTRAINT shipment_status_events_source_check
    CHECK (source IN ('webhook', 'reconciliation', 'manual'));
