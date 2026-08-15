-- +goose Up

ALTER TABLE batches ADD COLUMN reviewed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE batches DROP COLUMN reviewed_at;
