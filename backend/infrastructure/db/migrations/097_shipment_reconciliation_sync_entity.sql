-- +goose Up

-- Adds 'shipment_reconciliation' to sync_schedules.entity_type -- the
-- shipping module's own reconciliation schedule (see
-- domain/channels.SyncEntityShipmentReconciliation's doc comment), reusing
-- this generic per-connection schedule shape rather than a parallel
-- scheduling mechanism.
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation', 'shipment_reconciliation'));

-- +goose Down

ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation'));
