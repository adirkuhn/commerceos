-- +goose Up
-- New global cron job: every 24h, report stock close to expiry for each
-- tenant that turned the report on in Settings > Inventory (see
-- application/scheduler.RunStockExpiryReportUseCase). Enabled by default --
-- tenants opt in through their own setting, so a tenant that never does
-- costs one settings query.
INSERT INTO scheduled_jobs (job_key, enabled, interval_minutes) VALUES
    ('stock_expiry_report', true, 1440);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'stock_expiry_report';
