-- +goose Up

-- Tracks whether a SKU's cost_price_minor_units came from the daily
-- recalculation cron ('computed', the default) or a human manually setting
-- it (see application/pms's SetManualCostPriceUseCase). The cron skips any
-- SKU with source='manual' entirely, so a manual override sticks until a
-- human clears it (ClearCostPriceOverrideUseCase resets source back to
-- 'computed' -- the next cron run then recomputes the value).
ALTER TABLE skus ADD COLUMN cost_price_source TEXT NOT NULL DEFAULT 'computed';
ALTER TABLE skus ADD CONSTRAINT skus_cost_price_source_check
    CHECK (cost_price_source IN ('computed', 'manual'));

-- +goose Down
ALTER TABLE skus DROP CONSTRAINT skus_cost_price_source_check;
ALTER TABLE skus DROP COLUMN cost_price_source;
