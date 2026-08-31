-- +goose Up

-- CreateShipmentUseCase can optionally also register the sales order via
-- MyParcel's separate Orders/Fulfilment API (ExportMyParcelOrderUseCase) --
-- the only thing that makes it show up on MyParcel's own Orders dashboard,
-- which the plain Shipments API this codebase uses by default never
-- populates. Off by default: it hands MyParcel the full order (line items,
-- prices), which a merchant may not want to share just to print a label.
ALTER TABLE shipping_connection_settings ADD COLUMN export_order_to_myparcel BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE shipping_connection_settings DROP COLUMN export_order_to_myparcel;
