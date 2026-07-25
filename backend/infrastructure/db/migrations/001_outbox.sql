-- +goose Up

CREATE TABLE outbox_messages (
    id          BIGSERIAL PRIMARY KEY,
    event_id    UUID        NOT NULL,
    subject     TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    occurred_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT outbox_messages_event_id_unique UNIQUE (event_id),
    CONSTRAINT outbox_messages_status_check CHECK (status IN ('pending', 'published', 'failed'))
);

CREATE INDEX idx_outbox_pending ON outbox_messages (id) WHERE status = 'pending';

-- +goose Down

DROP TABLE IF EXISTS outbox_messages;
