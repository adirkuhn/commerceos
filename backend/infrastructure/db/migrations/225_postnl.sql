-- +goose Up
-- Direct PostNL shipping integration (provider "postnl"). The account
-- identifiers every PostNL call needs live on the connection; the API key
-- stays in consumer_secret_encrypted like every other provider.
ALTER TABLE integration_connections
    ADD COLUMN postnl_customer_code   TEXT    NOT NULL DEFAULT '',
    ADD COLUMN postnl_customer_number TEXT    NOT NULL DEFAULT '',
    ADD COLUMN postnl_sandbox         BOOLEAN NOT NULL DEFAULT FALSE;

-- PostNL returns a label only once, in the create response, so it is stored
-- (domain/shipping.ShipmentLabel). MyParcel labels are fetched on demand and
-- never stored here.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE shipment_labels (
    shipment_id  UUID        PRIMARY KEY REFERENCES shipments(id) ON DELETE CASCADE,
    tenant_id    UUID        NOT NULL,
    content_type TEXT        NOT NULL,
    data         BYTEA       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE shipment_labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipment_labels FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON shipment_labels
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS shipment_labels;
ALTER TABLE integration_connections
    DROP COLUMN IF EXISTS postnl_sandbox,
    DROP COLUMN IF EXISTS postnl_customer_number,
    DROP COLUMN IF EXISTS postnl_customer_code;
