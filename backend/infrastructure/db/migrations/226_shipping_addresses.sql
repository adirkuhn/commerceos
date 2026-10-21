-- +goose Up
-- The tenant's one sender address and one return address for labels
-- (domain/shipping.ShippingPreferences). NULL sender = use the company
-- profile address; NULL return = undeliverable parcels go back to the
-- sender.
ALTER TABLE shipping_preferences
    ADD COLUMN sender_address JSONB,
    ADD COLUMN return_address JSONB;

-- +goose Down
ALTER TABLE shipping_preferences
    DROP COLUMN IF EXISTS return_address,
    DROP COLUMN IF EXISTS sender_address;
