-- +goose Up

-- expiry_mismatch discrepancies used to bake the invoice expiry date into
-- details as YYYY-MM-DD text, which can't follow the tenant's date-format
-- setting. Store it structured; the UI formats it. NULL for every other
-- discrepancy type. Existing RLS policy on the table is unaffected.
ALTER TABLE receiving_discrepancies ADD COLUMN expiry_date DATE;

-- +goose Down
ALTER TABLE receiving_discrepancies DROP COLUMN expiry_date;
