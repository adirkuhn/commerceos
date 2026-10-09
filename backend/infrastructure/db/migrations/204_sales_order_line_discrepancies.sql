-- +goose Up
-- Differences between an imported order and the channel's own order (an
-- item missing, a different quantity, or the channel's lines not adding up
-- to its own total) -- see domain/oms.LineDiscrepancy. Only reported, on
-- the Stock Issues page: CommerceOS never changes the order itself.
-- One row per order, kind and item; re-imports update it in place.
CREATE TABLE sales_order_line_discrepancies (
    id                         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                  UUID        NOT NULL,
    sales_order_id             UUID        NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    kind                       TEXT        NOT NULL CHECK (kind IN ('missing', 'quantity', 'total')),
    item_key                   TEXT        NOT NULL,
    sku_code                   TEXT        NOT NULL DEFAULT '',
    name                       TEXT        NOT NULL DEFAULT '',
    channel_quantity           BIGINT      NOT NULL DEFAULT 0,
    local_quantity             BIGINT      NOT NULL DEFAULT 0,
    channel_total_minor_units  BIGINT      NOT NULL DEFAULT 0,
    local_total_minor_units    BIGINT      NOT NULL DEFAULT 0,
    detected_at                TIMESTAMPTZ NOT NULL,
    dismissed_at               TIMESTAMPTZ,
    resolved_at                TIMESTAMPTZ,
    UNIQUE (sales_order_id, kind, item_key)
);

CREATE INDEX sales_order_line_discrepancies_open_idx ON sales_order_line_discrepancies (tenant_id, detected_at DESC)
    WHERE dismissed_at IS NULL AND resolved_at IS NULL;

ALTER TABLE sales_order_line_discrepancies ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_line_discrepancies FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_order_line_discrepancies
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE IF EXISTS sales_order_line_discrepancies;
