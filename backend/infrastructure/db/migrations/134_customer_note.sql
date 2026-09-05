-- +goose Up
ALTER TABLE customers ADD COLUMN customer_note TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE customers DROP COLUMN customer_note;
