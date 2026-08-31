-- +goose Up

-- CancelShipmentUseCase's soft-cancel: staff can ask to delete a shipment
-- created with the wrong carrier. MyParcel's own DELETE only succeeds
-- pre-label-creation, so this column is our own gate, not MyParcel's --
-- once set, CreateShipmentUseCase's existing-shipment guard skips this row,
-- letting a corrected shipment be created for the same order. The row
-- itself (and its status history) is kept, not hard-deleted, for audit.
ALTER TABLE shipments ADD COLUMN cancelled_at TIMESTAMPTZ;
ALTER TABLE shipments ADD COLUMN cancel_note TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE shipments DROP COLUMN cancel_note;
ALTER TABLE shipments DROP COLUMN cancelled_at;
