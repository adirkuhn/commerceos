-- +goose Up

-- How the picking module works for the tenant: 'scan' (scan each item into a
-- picking session) or 'list' (see the order's items, mark the whole order
-- picked with one action). See domain/settings.PickingMode.
ALTER TABLE tenant_settings ADD COLUMN picking_mode TEXT NOT NULL DEFAULT 'scan'
    CONSTRAINT tenant_settings_picking_mode_check CHECK (picking_mode IN ('scan', 'list'));

-- +goose Down
ALTER TABLE tenant_settings DROP COLUMN picking_mode;
