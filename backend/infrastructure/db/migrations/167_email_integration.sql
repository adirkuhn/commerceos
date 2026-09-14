-- +goose Up

-- ProviderEmail-only columns (NULL for every other provider) -- see
-- domain/integrations.ProviderEmail's own doc comment for the full field
-- mapping. store_url/consumer_key/consumer_secret_encrypted already exist
-- and cover host/username/password.
ALTER TABLE integration_connections ADD COLUMN smtp_port TEXT;
ALTER TABLE integration_connections ADD COLUMN email_from TEXT;

-- +goose Down
ALTER TABLE integration_connections DROP COLUMN smtp_port;
ALTER TABLE integration_connections DROP COLUMN email_from;
