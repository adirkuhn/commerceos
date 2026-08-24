-- +goose Up

-- Bol gets its own entity_type values rather than reusing products/stock/
-- orders -- cmd/scheduler/dispatch.go's runner map is keyed by
-- (entity_type, direction) alone with exactly one runner per key, so
-- reusing Woo's keys would silently route a Bol connection's schedule
-- through the Woo-only runner. Same reasoning shipment_reconciliation
-- followed for MyParcel vs Woo's orders_reconciliation -- see
-- domain/channels.SyncEntityShipmentReconciliation's doc comment.
-- bol_products/bol_stock both dispatch to the same runner
-- (RunScheduledBolExportUseCase) on two separate schedule rows/intervals --
-- Bol has no stock-only endpoint, every push sends price+stock together.
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation', 'shipment_reconciliation', 'bol_orders', 'bol_products', 'bol_stock'));

-- +goose Down
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation', 'shipment_reconciliation'));
