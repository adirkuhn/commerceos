-- +goose Up

-- One row per sales order exported via MyParcel's Orders/Fulfilment API
-- (ExportMyParcelOrderUseCase) -- lets re-clicking "Create label" be
-- idempotent (look up an existing export before calling MyParcel again)
-- and gives the later shipment-create step something to look back at.
-- UNIQUE(connection_id, sales_order_id): one export per order per
-- connection, matching shipping_carrier_overrides' own
-- UNIQUE(connection_id, carrier) convention.
CREATE TABLE myparcel_fulfilment_orders (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    connection_id  UUID        NOT NULL,
    sales_order_id UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    myparcel_uuid  TEXT        NOT NULL,
    status         TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (connection_id, sales_order_id)
);

CREATE INDEX myparcel_fulfilment_orders_tenant_id_idx ON myparcel_fulfilment_orders (tenant_id);

ALTER TABLE myparcel_fulfilment_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE myparcel_fulfilment_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON myparcel_fulfilment_orders
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE myparcel_fulfilment_orders;
