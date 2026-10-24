-- +goose Up
-- A quote may be for a customer (domain/quotes.CustomerRef). Deleting the
-- customer keeps the quote, without one.
ALTER TABLE quotes ADD COLUMN customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;
CREATE INDEX quotes_customer_idx ON quotes (customer_id) WHERE customer_id IS NOT NULL;

-- +goose Down
ALTER TABLE quotes DROP COLUMN IF EXISTS customer_id;
