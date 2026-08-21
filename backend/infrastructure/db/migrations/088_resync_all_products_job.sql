-- +goose Up
-- New global cron job: periodically force-push every linked product's
-- current fields/stock to its channels, regardless of whether anything
-- changed since the last sync -- catches drift the event-driven paths
-- would miss (a channel-side edit, a missed/failed event that never
-- retried). Disabled by default -- opt-in per tenant scale, since walking
-- every linked product is real per-product Woo API traffic, unlike the
-- cheap sweepers seeded in 085_scheduled_jobs.sql.
INSERT INTO scheduled_jobs (job_key, enabled, interval_minutes) VALUES
    ('resync_all_products_to_channels', false, 1440);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'resync_all_products_to_channels';
