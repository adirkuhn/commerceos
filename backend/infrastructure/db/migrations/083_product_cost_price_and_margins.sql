-- +goose Up

-- Landed cost capture at receive time: a batch created against a specific
-- PO line item carries its own per-unit cost (that line's UnitCostMinorUnits
-- plus its proportional share of the PO's shipping/fee/other-charge lines --
-- see application/procurement's landed-cost allocation). Both nullable: not
-- every batch originates from a PO (manual stock-ins, POS-originated stock).
ALTER TABLE batches
    ADD COLUMN purchase_order_line_item_id UUID REFERENCES purchase_order_line_items(id) ON DELETE SET NULL,
    ADD COLUMN landed_cost_minor_units BIGINT;

-- SKU's rolled-up cost price -- the weighted average of on-hand batches'
-- landed cost, recomputed daily by the cron (see
-- application/pms.RecalculateCostPricesUseCase). Nil until the cron has run
-- at least once with a cost basis to average.
ALTER TABLE skus ADD COLUMN cost_price_minor_units BIGINT;

-- Per-category target-margin override; nil means "use the tenant default"
-- (tenant_settings.default_target_margin_basis_points below). Basis points,
-- 0-9999 (a margin strictly under 100%).
ALTER TABLE categories
    ADD COLUMN target_margin_basis_points INTEGER,
    ADD CONSTRAINT categories_target_margin_basis_points_check
        CHECK (target_margin_basis_points IS NULL OR (target_margin_basis_points >= 0 AND target_margin_basis_points < 10000));

-- Tenant-wide fallback target margin. 0 (the default) means "unset" on a
-- never-saved tenant, same zero-value convention as this table's other
-- columns -- not "0% margin".
ALTER TABLE tenant_settings
    ADD COLUMN default_target_margin_basis_points INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT tenant_settings_default_target_margin_basis_points_check
        CHECK (default_target_margin_basis_points >= 0 AND default_target_margin_basis_points < 10000);

-- Daily snapshot of a SKU's computed cost price and recommended price --
-- the margin-trend-over-time chart's data source. One row per SKU per
-- calendar day the cron computed a value for it; the unique constraint is
-- what makes a same-day cron rerun idempotent (see
-- pms.SKUCostPriceHistoryRepository.Upsert).
CREATE TABLE sku_cost_price_history (
    id                            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                     UUID        NOT NULL,
    sku_id                        UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    cost_price_minor_units        BIGINT      NOT NULL,
    recommended_price_minor_units BIGINT      NOT NULL,
    occurred_on                   DATE        NOT NULL,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT sku_cost_price_history_sku_id_occurred_on_key UNIQUE (sku_id, occurred_on)
);

CREATE INDEX sku_cost_price_history_tenant_id_sku_id_idx ON sku_cost_price_history (tenant_id, sku_id, occurred_on DESC);

ALTER TABLE sku_cost_price_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE sku_cost_price_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sku_cost_price_history
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- +goose Down
DROP TABLE sku_cost_price_history;
ALTER TABLE tenant_settings
    DROP CONSTRAINT tenant_settings_default_target_margin_basis_points_check,
    DROP COLUMN default_target_margin_basis_points;
ALTER TABLE categories
    DROP CONSTRAINT categories_target_margin_basis_points_check,
    DROP COLUMN target_margin_basis_points;
ALTER TABLE skus DROP COLUMN cost_price_minor_units;
ALTER TABLE batches
    DROP COLUMN landed_cost_minor_units,
    DROP COLUMN purchase_order_line_item_id;
