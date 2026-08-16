-- +goose Up

ALTER TABLE outbox_messages ADD COLUMN retry_count INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE outbox_messages DROP COLUMN retry_count;
