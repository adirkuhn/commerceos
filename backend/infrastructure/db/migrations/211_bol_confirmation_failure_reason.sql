-- +goose Up

-- Why a Bol shipment confirmation failed (domain/channels BolFailure*), so
-- the admin can explain it in the reader's language. '' for everything
-- else, including rows from before this existed.
ALTER TABLE bol_shipment_confirmations ADD COLUMN failure_reason TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bol_shipment_confirmations DROP COLUMN failure_reason;
