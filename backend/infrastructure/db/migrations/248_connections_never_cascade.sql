-- +goose Up
-- COM-22: a connection row is the store's identity. Deleting one by accident
-- on 2026-11-03 took 664 product links, the sales channel and its schedules
-- with it. The app stopped deleting these rows in COM-15, so nothing relies
-- on the cascade: a raw delete now fails instead of wiping them.
ALTER TABLE external_category_links DROP CONSTRAINT external_category_links_connection_id_fkey;
ALTER TABLE external_category_links ADD CONSTRAINT external_category_links_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE import_jobs DROP CONSTRAINT import_jobs_connection_id_fkey;
ALTER TABLE import_jobs ADD CONSTRAINT import_jobs_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE moneybird_contacts DROP CONSTRAINT moneybird_contacts_connection_id_fkey;
ALTER TABLE moneybird_contacts ADD CONSTRAINT moneybird_contacts_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE moneybird_documents DROP CONSTRAINT moneybird_documents_connection_id_fkey;
ALTER TABLE moneybird_documents ADD CONSTRAINT moneybird_documents_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE moneybird_settings DROP CONSTRAINT moneybird_settings_connection_id_fkey;
ALTER TABLE moneybird_settings ADD CONSTRAINT moneybird_settings_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE product_channel_listings DROP CONSTRAINT product_channel_listings_connection_id_fkey;
ALTER TABLE product_channel_listings ADD CONSTRAINT product_channel_listings_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE sales_channels DROP CONSTRAINT sales_channels_connection_id_fkey;
ALTER TABLE sales_channels ADD CONSTRAINT sales_channels_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_connection_id_fkey;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE external_category_links DROP CONSTRAINT external_category_links_connection_id_fkey;
ALTER TABLE external_category_links ADD CONSTRAINT external_category_links_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE import_jobs DROP CONSTRAINT import_jobs_connection_id_fkey;
ALTER TABLE import_jobs ADD CONSTRAINT import_jobs_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE moneybird_contacts DROP CONSTRAINT moneybird_contacts_connection_id_fkey;
ALTER TABLE moneybird_contacts ADD CONSTRAINT moneybird_contacts_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE moneybird_documents DROP CONSTRAINT moneybird_documents_connection_id_fkey;
ALTER TABLE moneybird_documents ADD CONSTRAINT moneybird_documents_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE moneybird_settings DROP CONSTRAINT moneybird_settings_connection_id_fkey;
ALTER TABLE moneybird_settings ADD CONSTRAINT moneybird_settings_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE product_channel_listings DROP CONSTRAINT product_channel_listings_connection_id_fkey;
ALTER TABLE product_channel_listings ADD CONSTRAINT product_channel_listings_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE sales_channels DROP CONSTRAINT sales_channels_connection_id_fkey;
ALTER TABLE sales_channels ADD CONSTRAINT sales_channels_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
ALTER TABLE sync_schedules DROP CONSTRAINT sync_schedules_connection_id_fkey;
ALTER TABLE sync_schedules ADD CONSTRAINT sync_schedules_connection_id_fkey
    FOREIGN KEY (connection_id) REFERENCES integration_connections(id) ON DELETE CASCADE;
