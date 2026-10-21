-- +goose Up
-- The tenant's shipping choices that span connections (domain/shipping.
-- ShippingPreferences): which shipping connection packing uses unless
-- someone picks another one for a label. NULL default_connection_id means
-- "not chosen" -- ResolveDefaultConnection then uses the oldest shipping
-- connection. Deleting that connection clears the choice.
-- commerceos_app gets its grants from 020's ALTER DEFAULT PRIVILEGES.
CREATE TABLE shipping_preferences (
    tenant_id             UUID        PRIMARY KEY,
    default_connection_id UUID        REFERENCES integration_connections(id) ON DELETE SET NULL,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE shipping_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_preferences FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON shipping_preferences
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS shipping_preferences;
