-- +goose Up
-- Global job: every hour, re-push order statuses WooCommerce missed (the
-- status push gives up after a few failed deliveries, so a store outage
-- left Woo on processing for shipped orders -- 2026-10-31..11-03). Cheap:
-- one Woo call per 100 recent shipped/completed orders, pushes only where
-- Woo is behind. See application/channels.RepushWooOrderStatusesUseCase.
INSERT INTO scheduled_jobs (job_key, enabled, interval_minutes) VALUES
    ('repush_woo_order_statuses', true, 60);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'repush_woo_order_statuses';
