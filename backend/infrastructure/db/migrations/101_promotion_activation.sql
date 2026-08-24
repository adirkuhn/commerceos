-- +goose Up

-- Tracks whether a promotion_skus line's discount is currently live on its
-- SKU's sale price, and what to restore when the parent promotion leaves
-- its active window -- see domain/promotions.PromotionSKU's doc.
ALTER TABLE promotion_skus ADD COLUMN applied BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE promotion_skus ADD COLUMN previous_sale_price_minor_units BIGINT;
ALTER TABLE promotion_skus ADD COLUMN previous_sale_end_date DATE;

-- Speeds up the periodic activation/reversal sweep's "not yet applied" /
-- "currently applied" filters.
CREATE INDEX promotion_skus_applied_idx ON promotion_skus (applied);

-- New global cron job: activate/revert promotion-driven sale prices --
-- see application/scheduler.RunPromotionActivationUseCase. Same 1-minute
-- cadence as the other cheap sweepers seeded in 085_scheduled_jobs.sql.
INSERT INTO scheduled_jobs (job_key, interval_minutes) VALUES
    ('promotion_activation', 1);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'promotion_activation';
DROP INDEX promotion_skus_applied_idx;
ALTER TABLE promotion_skus DROP COLUMN previous_sale_end_date;
ALTER TABLE promotion_skus DROP COLUMN previous_sale_price_minor_units;
ALTER TABLE promotion_skus DROP COLUMN applied;
