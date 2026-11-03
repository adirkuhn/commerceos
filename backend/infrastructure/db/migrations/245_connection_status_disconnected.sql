-- +goose Up
-- COM-15: disconnecting an integration no longer deletes its row (the FKs
-- cascaded into sales channels, listings, schedules and category links and
-- nulled sales_orders.connection_id). The row stays with status
-- 'disconnected' and wiped credentials; reconnecting the same store takes
-- it back. The previous release never writes this value, so it still runs.
ALTER TABLE integration_connections DROP CONSTRAINT integration_connections_status_check;
ALTER TABLE integration_connections ADD CONSTRAINT integration_connections_status_check
    CHECK (status = ANY (ARRAY['connected', 'error', 'disconnected']));

-- +goose Down
UPDATE integration_connections SET status = 'error' WHERE status = 'disconnected';
ALTER TABLE integration_connections DROP CONSTRAINT integration_connections_status_check;
ALTER TABLE integration_connections ADD CONSTRAINT integration_connections_status_check
    CHECK (status = ANY (ARRAY['connected', 'error']));
