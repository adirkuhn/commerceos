-- +goose Up

-- Consumer-side idempotency ledger: JetStream is at-least-once delivery, so
-- every consumer checks this before running its side effects. Not
-- tenant-scoped data (it's event-processing bookkeeping, not business
-- data), so no tenant_id/RLS here — matches the outbox table.
CREATE TABLE processed_events (
    event_id     UUID        NOT NULL,
    consumer     TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, consumer)
);

-- +goose Down
DROP TABLE processed_events;
