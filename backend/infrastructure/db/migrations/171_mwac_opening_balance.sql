-- +goose Up

-- Phase 4 of the MWAC migration: give every existing SKU a clean opening
-- state instead of trying to reconstruct blended cost history from old
-- purchase orders. See sku_cost_price_ledger's doc comment (migration 169)
-- and pms.SKUCostPriceSourceUnknown's.

-- A SKU whose cost has genuinely never been computed (cost_price_source
-- still the 'computed' default, but no on-hand batch ever gave the old
-- cron a cost basis to average, so cost_price_minor_units is nil) becomes
-- explicitly 'unknown' -- distinguishes "no cost basis exists at all" from
-- "computed, just hasn't run since a schema change". A SKU the old cron
-- DID successfully compute a value for keeps that value/source as-is: it's
-- a legitimate opening balance, not a gap.
UPDATE skus
SET cost_price_source = 'unknown'
WHERE cost_price_source = 'computed' AND cost_price_minor_units IS NULL;

-- One opening-balance ledger row per active SKU with positive on-hand
-- quantity (summed from the existing stock_movements ledger -- see
-- inventory.Batch's doc comment: current quantity is never stored
-- directly). cost_price_minor_units/inventory_value_minor_units stay NULL
-- for an 'unknown' SKU -- never a fabricated 0, same policy as everywhere
-- else in this feature.
INSERT INTO sku_cost_price_ledger (
    tenant_id, sku_id, source_type, quantity_delta, quantity_after,
    cost_price_minor_units, inventory_value_minor_units, occurred_at
)
SELECT
    s.tenant_id,
    s.id,
    'opening',
    qty.total_quantity,
    qty.total_quantity,
    s.cost_price_minor_units,
    CASE WHEN s.cost_price_minor_units IS NOT NULL THEN qty.total_quantity * s.cost_price_minor_units END,
    now()
FROM skus s
JOIN (
    SELECT sku_id, SUM(quantity_delta) AS total_quantity
    FROM stock_movements
    GROUP BY sku_id
    HAVING SUM(quantity_delta) > 0
) qty ON qty.sku_id = s.id
WHERE s.status = 'active';

-- +goose Down

-- Does not revert cost_price_source back to 'computed' for rows the Up
-- step flipped to 'unknown' -- indistinguishable from 'unknown' rows
-- created by ordinary activity after Up ran, so reverting them would be
-- guessing. A rollback of this migration is a one-way data cleanup, not
-- meant to be safe to run back-and-forth in production.
DELETE FROM sku_cost_price_ledger WHERE source_type = 'opening';
