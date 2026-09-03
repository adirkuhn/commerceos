-- +goose Up
ALTER TABLE product_channel_listings ADD COLUMN gated_at TIMESTAMPTZ NULL;

-- +goose Down
ALTER TABLE product_channel_listings DROP COLUMN gated_at;
