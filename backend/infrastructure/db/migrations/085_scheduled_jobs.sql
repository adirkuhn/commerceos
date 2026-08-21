-- +goose Up
-- Global, cross-tenant recurring jobs -- the "cost price recalculation",
-- "expired sale prices", "expired batches" sweepers cmd/scheduler already
-- ran on its own ticker with no persisted state at all (last-run
-- success/failure was only ever visible in process logs). No tenant_id --
-- these are platform-operational jobs, one run touches every tenant, same
-- reasoning as outbox_messages having none. Deliberately NOT RLS'd: only
-- reachable through the super_admin-gated admin API and the scheduler
-- process itself, neither of which is tenant-scoped.
CREATE TABLE scheduled_jobs (
    id                BIGSERIAL   PRIMARY KEY,
    job_key           TEXT        NOT NULL,
    enabled           BOOLEAN     NOT NULL DEFAULT true,
    interval_minutes  INTEGER     NOT NULL,
    last_run_at       TIMESTAMPTZ,
    last_run_status   TEXT        NOT NULL DEFAULT '',
    last_run_error    TEXT        NOT NULL DEFAULT '',
    next_run_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scheduled_jobs_job_key_unique UNIQUE (job_key),
    CONSTRAINT scheduled_jobs_last_run_status_check CHECK (last_run_status IN ('', 'success', 'error'))
);

-- Seeded rows for the jobs cmd/scheduler already runs -- expired_sale_prices/
-- expired_batches are cheap sweepers, effectively continuous (1 minute);
-- cost_price_recalculation is real per-SKU work, once a day.
INSERT INTO scheduled_jobs (job_key, interval_minutes) VALUES
    ('expired_sale_prices', 1),
    ('expired_batches', 1),
    ('cost_price_recalculation', 1440);

-- +goose Down
DROP TABLE IF EXISTS scheduled_jobs;
