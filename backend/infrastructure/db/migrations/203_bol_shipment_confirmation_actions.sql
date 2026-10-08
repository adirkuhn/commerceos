-- +goose Up
-- A Bol shipment confirmation is now one of two Bol requests (see
-- domain/channels.BolConfirmationAction): marking order items shipped
-- (create_shipment, every row so far), or correcting the transporter +
-- track & trace of a shipment Bol already has after its parcel was
-- cancelled and replaced (update_transport, with Bol's transport_id).
ALTER TABLE bol_shipment_confirmations
    ADD COLUMN action TEXT NOT NULL DEFAULT 'create_shipment'
        CHECK (action IN ('create_shipment', 'update_transport')),
    ADD COLUMN transport_id TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bol_shipment_confirmations DROP COLUMN transport_id, DROP COLUMN action;
