-- +goose Up
ALTER TABLE receiving_discrepancies ADD COLUMN status TEXT NOT NULL DEFAULT 'open';
ALTER TABLE receiving_discrepancies ADD COLUMN acknowledged_at TIMESTAMPTZ;

ALTER TABLE receiving_discrepancies DROP CONSTRAINT receiving_discrepancies_type_check;
ALTER TABLE receiving_discrepancies ADD CONSTRAINT receiving_discrepancies_type_check
    CHECK (type IN ('quantity_mismatch', 'missing_item', 'unexpected_item', 'expiry_mismatch', 'damaged_item'));

ALTER TABLE receiving_discrepancies ADD CONSTRAINT receiving_discrepancies_status_check
    CHECK (status IN ('open', 'acknowledged'));

-- +goose Down
ALTER TABLE receiving_discrepancies DROP CONSTRAINT receiving_discrepancies_status_check;

ALTER TABLE receiving_discrepancies DROP CONSTRAINT receiving_discrepancies_type_check;
ALTER TABLE receiving_discrepancies ADD CONSTRAINT receiving_discrepancies_type_check
    CHECK (type IN ('quantity_mismatch', 'missing_item', 'unexpected_item', 'expiry_mismatch'));

ALTER TABLE receiving_discrepancies DROP COLUMN acknowledged_at;
ALTER TABLE receiving_discrepancies DROP COLUMN status;
