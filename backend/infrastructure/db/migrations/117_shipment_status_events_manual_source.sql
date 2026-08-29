-- +goose Up

-- "Refresh from provider" (RefreshShipmentTrackingUseCase) writes a manual
-- entry to the same history table webhook/reconciliation already write to
-- -- widen the source check constraint to allow it.
ALTER TABLE shipment_status_events DROP CONSTRAINT shipment_status_events_source_check;
ALTER TABLE shipment_status_events ADD CONSTRAINT shipment_status_events_source_check
    CHECK (source IN ('webhook', 'reconciliation', 'manual'));

-- The tracking-timeline read (ListShipmentTrackingEventsUseCase) is
-- "WHERE shipment_id = ? ORDER BY occurred_at DESC" -- the existing
-- single-column shipment_id index can filter but still needs a separate
-- sort step; this composite index covers both in one pass.
CREATE INDEX shipment_status_events_shipment_id_occurred_at_idx
    ON shipment_status_events (shipment_id, occurred_at DESC);

-- +goose Down
DROP INDEX shipment_status_events_shipment_id_occurred_at_idx;
ALTER TABLE shipment_status_events DROP CONSTRAINT shipment_status_events_source_check;
ALTER TABLE shipment_status_events ADD CONSTRAINT shipment_status_events_source_check
    CHECK (source IN ('webhook', 'reconciliation'));
