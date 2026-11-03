-- +goose Up
-- COM-18: the WooCommerce products and stock sync schedules (import and
-- export) are gone -- changes reach the store through events, and a full
-- push every interval flooded Woo (182 PUTs a minute). Their rows go; a
-- new stock_reconciliation schedule replaces the safety net, pushing only
-- the listings whose Woo stock drifted. 'products' and 'stock' stay legal
-- in the CHECK for now so the previous release keeps working during a
-- deploy (the application no longer accepts them).
DELETE FROM sync_schedules WHERE entity_type IN ('products', 'stock');

ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation', 'shipment_reconciliation',
        'bol_orders', 'bol_products', 'bol_stock', 'stock_reconciliation'));

-- +goose Down
DELETE FROM sync_schedules WHERE entity_type = 'stock_reconciliation';
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_entity_type_check;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_entity_type_check
    CHECK (entity_type IN ('products', 'stock', 'orders', 'categories', 'orders_reconciliation', 'shipment_reconciliation', 'bol_orders', 'bol_products', 'bol_stock'));
