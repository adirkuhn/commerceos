-- +goose Up

-- orders_reconciliation is a second, independent sync schedule for the
-- same connection -- a daily pass over a fixed rolling window (see
-- cmd/scheduler/dispatch.go's reconciliationWindow), re-run through the
-- same idempotent (dedupes by ExternalID) ImportWooOrdersUseCase the
-- regular "orders" schedule uses. It exists to catch any order the
-- incremental "orders" schedule missed, from whatever cause, without
-- needing separate diff/detection logic.
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation'));

-- +goose Down
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories'));
