-- +goose Up

-- The structured outcome of cancelling a shipment (see
-- domain/shipping.CancelOutcome), so the admin can show it in the reader's
-- language. NULL on shipments cancelled before this existed -- they keep
-- only their English cancel_note.
ALTER TABLE shipments
    ADD COLUMN cancel_deleted_on_carrier BOOLEAN,
    ADD COLUMN cancel_carrier_error TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE shipments
    DROP COLUMN cancel_carrier_error,
    DROP COLUMN cancel_deleted_on_carrier;
