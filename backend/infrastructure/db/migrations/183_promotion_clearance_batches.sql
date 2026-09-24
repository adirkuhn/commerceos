-- +goose Up

-- Clearance lines: a promotion_skus row can sell off one near-expiry batch
-- and end on its own once that batch is sold out, discarded, removed or
-- expired -- see domain/promotions.PromotionSKU's BatchID/FinishedAt.
--
-- No FK to batches on purpose: a batch being deleted is one of the ways a
-- clearance ends, and the line must survive that (a RESTRICT FK would block
-- the delete, a CASCADE/SET NULL one would silently drop or un-clearance the
-- line before the sweep can end it and restore the SKU's price).
ALTER TABLE promotion_skus ADD COLUMN batch_id UUID;
ALTER TABLE promotion_skus ADD COLUMN finished_at TIMESTAMPTZ;

-- The clearance sweep and stock-changed consumer only ever look at open
-- clearance lines.
CREATE INDEX promotion_skus_open_clearance_idx ON promotion_skus (sku_id)
    WHERE batch_id IS NOT NULL AND finished_at IS NULL;

-- +goose Down
DROP INDEX promotion_skus_open_clearance_idx;
ALTER TABLE promotion_skus DROP COLUMN finished_at;
ALTER TABLE promotion_skus DROP COLUMN batch_id;
