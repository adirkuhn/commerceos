-- +goose Up
-- A sync schedule's failure streak, so staff are alerted once when it keeps
-- failing (2026-10-31..11-03 a Woo 403 went unnoticed for three days) and
-- once when it recovers. Additive with defaults: older binaries ignore it.
ALTER TABLE sync_schedules
    ADD COLUMN failing_since   TIMESTAMPTZ,
    ADD COLUMN failure_alerted BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE sync_schedules
    DROP COLUMN failure_alerted,
    DROP COLUMN failing_since;
