-- +goose Up

-- Third cost_price_source state: a SKU whose existing on-hand quantity has
-- no reliable historical cost to carry forward (see the MWAC-migration
-- opening-balance policy). Distinct from 'computed' (the cron has a real
-- cost basis, just hasn't run yet / value is nil transiently) -- 'unknown'
-- means "no cost basis exists at all", so CostPriceMinorUnits stays nil and
-- sales against this SKU record no COGS until the first real receipt gives
-- it a cost (see application/pms's moving-average-cost merge behavior for
-- a SKU currently unknown).
ALTER TABLE skus DROP CONSTRAINT skus_cost_price_source_check;
ALTER TABLE skus ADD CONSTRAINT skus_cost_price_source_check
    CHECK (cost_price_source IN ('computed', 'manual', 'unknown'));

-- Append-only ledger of a SKU's blended moving-average cost over time --
-- one immutable row per event that changes (or would change) it: opening
-- balance, each receipt, each sale, each stock adjustment/damage/loss/
-- return, each manual cost correction. Never updated or deleted; this is
-- what item 12 (per-item cost history) and item 10 (ledger-vs-snapshot
-- reconciliation) read from. Keyed by (sku_id, occurred_at, id) rather than
-- by calendar day -- unlike sku_cost_price_history's daily-snapshot grain,
-- multiple events on the same SKU on the same day each get their own row.
--
-- cost_price_minor_units is nullable: a SKU with cost_price_source =
-- 'unknown' writes rows here too (so "there were 100 units, cost unknown,
-- as of this date" is still part of the history), just with cost/value nil
-- rather than the [wrong] 0/1 that would silently contaminate valuation
-- reports (see the MWAC migration's explicit warning against that).
CREATE TABLE sku_cost_price_ledger (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    sku_id                    UUID        NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
    source_type               TEXT        NOT NULL,
    reference_type            TEXT,
    reference_id              UUID,
    quantity_delta            BIGINT      NOT NULL,
    quantity_after             BIGINT      NOT NULL,
    cost_price_minor_units    BIGINT,
    inventory_value_minor_units BIGINT,
    occurred_at               TIMESTAMPTZ NOT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT sku_cost_price_ledger_source_type_check
        CHECK (source_type IN ('opening', 'receipt', 'sale', 'return', 'damaged', 'lost', 'adjustment', 'correction'))
);

CREATE INDEX sku_cost_price_ledger_tenant_id_sku_id_idx ON sku_cost_price_ledger (tenant_id, sku_id, occurred_at DESC);

ALTER TABLE sku_cost_price_ledger ENABLE ROW LEVEL SECURITY;
ALTER TABLE sku_cost_price_ledger FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sku_cost_price_ledger
    USING (current_setting('app.current_tenant', true) = 'bypass_rls'
        OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid);

-- Cost frozen at the moment of sale -- never recomputed when the SKU's
-- current cost_price_minor_units later changes (see sale-path design: a
-- historical sale's margin must not drift because inventory cost moved).
-- Both nullable: a sale against a SKU with cost_price_source = 'unknown'
-- at that instant records no COGS rather than a fabricated 0.
ALTER TABLE sales_order_line_items
    ADD COLUMN cost_at_sale_minor_units BIGINT,
    ADD COLUMN cogs_minor_units BIGINT;

-- +goose Down
ALTER TABLE sales_order_line_items
    DROP COLUMN cogs_minor_units,
    DROP COLUMN cost_at_sale_minor_units;
DROP TABLE sku_cost_price_ledger;
ALTER TABLE skus DROP CONSTRAINT skus_cost_price_source_check;
ALTER TABLE skus ADD CONSTRAINT skus_cost_price_source_check
    CHECK (cost_price_source IN ('computed', 'manual'));
