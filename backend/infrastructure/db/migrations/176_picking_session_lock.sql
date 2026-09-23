-- +goose Up
ALTER TABLE picking_sessions ADD COLUMN locked_by TEXT;
ALTER TABLE picking_sessions ADD COLUMN locked_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE picking_sessions DROP COLUMN locked_at;
ALTER TABLE picking_sessions DROP COLUMN locked_by;
