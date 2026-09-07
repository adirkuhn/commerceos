-- +goose Up
-- A storefront key can now be scoped to one specific sales channel rather
-- than the whole tenant -- needed once a tenant can have more than one
-- internal storefront (e.g. a B2C channel and a B2B channel) with
-- different catalogs/pricing: the checkout API must know which channel's
-- rules apply, not just which tenant. NULL keeps today's behavior (a
-- tenant-wide key, e.g. the WooCommerce plugin's) unchanged. ON DELETE
-- CASCADE: deleting a sales channel a key was scoped to leaves no
-- meaningful key behind.
ALTER TABLE storefront_api_keys ADD COLUMN sales_channel_id UUID REFERENCES sales_channels(id) ON DELETE CASCADE;

CREATE INDEX storefront_api_keys_sales_channel_id_idx ON storefront_api_keys (sales_channel_id);

-- +goose Down
DROP INDEX IF EXISTS storefront_api_keys_sales_channel_id_idx;
ALTER TABLE storefront_api_keys DROP COLUMN sales_channel_id;
