-- +goose Up

ALTER TABLE batches
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'expired'));

-- +goose Down
ALTER TABLE batches DROP COLUMN status;
