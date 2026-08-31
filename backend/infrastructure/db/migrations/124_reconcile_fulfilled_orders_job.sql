-- +goose Up
-- New global cron job: every 24h, catch any SalesOrder whose shipment
-- already fulfilled it (Fulfilled=true) but whose Status never made it to
-- Shipped/OnHold -- self-heals whatever HandleShipmentWebhookUseCase/
-- HandleOrderStatusWebhookUseCase's webhook paths ever missed (see
-- application/scheduler.ReconcileFulfilledOrdersUseCase). Enabled by
-- default -- a cheap, bounded sweep (up to 500 orders/run), same class as
-- 085_scheduled_jobs.sql's original sweepers, not an opt-in per-tenant cost
-- like 088's resync job.
INSERT INTO scheduled_jobs (job_key, enabled, interval_minutes) VALUES
    ('reconcile_fulfilled_orders', true, 1440);

-- +goose Down
DELETE FROM scheduled_jobs WHERE job_key = 'reconcile_fulfilled_orders';
