-- +goose Up

-- One row per shipment created for a SalesOrder via MyParcel. The
-- recipient_* fields are a confirmed, MyParcel-shaped snapshot taken at
-- label-creation time -- deliberately NOT derived live from
-- sales_orders.shipping_* (which is Woo-shaped free text, no street/number
-- split) on every read; staff review/edit them once in the address-confirm
-- step, then this is what actually gets submitted and printed.
CREATE TABLE shipments (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID        NOT NULL,
    sales_order_id         UUID        NOT NULL,
    connection_id          UUID        NOT NULL,
    carrier                INTEGER     NOT NULL,
    myparcel_shipment_id   BIGINT,
    barcode                TEXT        NOT NULL DEFAULT '',
    status                 INTEGER     NOT NULL DEFAULT 0,
    status_main            TEXT        NOT NULL DEFAULT '',
    label_format           TEXT        NOT NULL DEFAULT 'pdf',
    package_type           INTEGER     NOT NULL DEFAULT 1,
    signature_required     BOOLEAN     NOT NULL DEFAULT FALSE,
    only_recipient         BOOLEAN     NOT NULL DEFAULT FALSE,
    insurance_amount_minor_units INTEGER,
    recipient_person       TEXT        NOT NULL,
    recipient_company      TEXT        NOT NULL DEFAULT '',
    recipient_street       TEXT        NOT NULL,
    recipient_number       TEXT        NOT NULL DEFAULT '',
    recipient_number_suffix TEXT       NOT NULL DEFAULT '',
    recipient_postal_code  TEXT        NOT NULL DEFAULT '',
    recipient_city         TEXT        NOT NULL,
    recipient_cc           TEXT        NOT NULL,
    recipient_email        TEXT        NOT NULL DEFAULT '',
    recipient_phone        TEXT        NOT NULL DEFAULT '',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipments_label_format_check CHECK (label_format IN ('pdf', 'zpl'))
);

CREATE INDEX shipments_tenant_id_idx ON shipments (tenant_id);
CREATE INDEX shipments_sales_order_id_idx ON shipments (sales_order_id);
-- Drives the reconciliation scheduler's "non-terminal, no recent update"
-- scan (see application/shipping.ReconcileShipmentsUseCase).
CREATE INDEX shipments_status_updated_at_idx ON shipments (status, updated_at);

ALTER TABLE shipments ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON shipments
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Append-only audit/history of every status MyParcel has reported for a
-- shipment, whether learned via webhook or the reconciliation poll --
-- shipments.status/status_main only ever holds the latest.
CREATE TABLE shipment_status_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    shipment_id  UUID        NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    status       INTEGER     NOT NULL,
    status_main  TEXT        NOT NULL DEFAULT '',
    source       TEXT        NOT NULL DEFAULT 'webhook',
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipment_status_events_source_check CHECK (source IN ('webhook', 'reconciliation'))
);

CREATE INDEX shipment_status_events_tenant_id_idx ON shipment_status_events (tenant_id);
CREATE INDEX shipment_status_events_shipment_id_idx ON shipment_status_events (shipment_id);

ALTER TABLE shipment_status_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipment_status_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON shipment_status_events
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Per-connection global shipping defaults + optional per-carrier override
-- rows -- "configure general configuration and settings global and per
-- carrier" from the design. One settings row per MyParcel connection;
-- carrier_overrides is a separate table since it's zero-or-many per
-- connection (one row per carrier the tenant wants to override).
CREATE TABLE shipping_connection_settings (
    connection_id       UUID PRIMARY KEY,
    tenant_id           UUID        NOT NULL,
    default_carrier     INTEGER     NOT NULL DEFAULT 1,
    default_package_type INTEGER    NOT NULL DEFAULT 1,
    default_label_format TEXT       NOT NULL DEFAULT 'pdf',
    default_signature_required BOOLEAN NOT NULL DEFAULT FALSE,
    default_only_recipient     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_connection_settings_label_format_check CHECK (default_label_format IN ('pdf', 'zpl'))
);

CREATE INDEX shipping_connection_settings_tenant_id_idx ON shipping_connection_settings (tenant_id);

ALTER TABLE shipping_connection_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_connection_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON shipping_connection_settings
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

CREATE TABLE shipping_carrier_overrides (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    connection_id    UUID        NOT NULL REFERENCES shipping_connection_settings(connection_id) ON DELETE CASCADE,
    carrier          INTEGER     NOT NULL,
    package_type     INTEGER,
    signature_required BOOLEAN,
    only_recipient   BOOLEAN,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (connection_id, carrier)
);

CREATE INDEX shipping_carrier_overrides_tenant_id_idx ON shipping_carrier_overrides (tenant_id);

ALTER TABLE shipping_carrier_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_carrier_overrides FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON shipping_carrier_overrides
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE shipping_carrier_overrides;
DROP TABLE shipping_connection_settings;
DROP TABLE shipment_status_events;
DROP TABLE shipments;
