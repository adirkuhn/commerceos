-- +goose Up

-- channel_sync_logs is an audit trail: it deliberately denormalizes
-- product_id/connection_id/provider/external_id off product_channel_listings
-- so history reads without a join. But listing_id was ON DELETE CASCADE,
-- which wiped that entire audit trail the moment a listing was deleted
-- (e.g. de-listing a product from a channel) -- defeating the point of
-- keeping the denormalized snapshot columns in the first place.
ALTER TABLE channel_sync_logs ALTER COLUMN listing_id DROP NOT NULL;
ALTER TABLE channel_sync_logs DROP CONSTRAINT channel_sync_logs_listing_id_fkey;
ALTER TABLE channel_sync_logs ADD CONSTRAINT channel_sync_logs_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES product_channel_listings(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE channel_sync_logs DROP CONSTRAINT channel_sync_logs_listing_id_fkey;
ALTER TABLE channel_sync_logs ADD CONSTRAINT channel_sync_logs_listing_id_fkey
    FOREIGN KEY (listing_id) REFERENCES product_channel_listings(id) ON DELETE CASCADE;
ALTER TABLE channel_sync_logs ALTER COLUMN listing_id SET NOT NULL;
