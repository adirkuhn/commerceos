-- +goose Up
-- COM-18: a listing whose product no longer exists on the channel (e.g.
-- deleted in WooCommerce) is marked 'missing' instead of being retried
-- forever. Additive: older code never writes it.
ALTER TABLE product_channel_listings DROP CONSTRAINT product_channel_listings_sync_status_check;
ALTER TABLE product_channel_listings ADD CONSTRAINT product_channel_listings_sync_status_check
    CHECK (sync_status IN ('pending', 'synced', 'error', 'missing'));

-- +goose Down
UPDATE product_channel_listings SET sync_status = 'error' WHERE sync_status = 'missing';
ALTER TABLE product_channel_listings DROP CONSTRAINT product_channel_listings_sync_status_check;
ALTER TABLE product_channel_listings ADD CONSTRAINT product_channel_listings_sync_status_check
    CHECK (sync_status IN ('pending', 'synced', 'error'));
