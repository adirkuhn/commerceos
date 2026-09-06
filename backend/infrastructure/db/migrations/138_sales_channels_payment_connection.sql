-- +goose Up
-- Which connected payment integration (Stripe/Revolut) this channel
-- generates a payment link through -- nil/NULL means no payment method
-- enabled yet. ON DELETE SET NULL: deleting the connection just disables
-- payment on this channel, it doesn't block the delete or orphan the
-- channel.
ALTER TABLE sales_channels ADD COLUMN payment_connection_id UUID REFERENCES integration_connections(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE sales_channels DROP COLUMN payment_connection_id;
