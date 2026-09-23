-- +goose Up

-- locked_since is when the current lock holder acquired the lock. Unlike
-- locked_at (bumped by every heartbeat) it answers "how long has this person
-- had the order" -- the packing queue shows it. Backfilled from locked_at,
-- the best approximation for locks held right now.
ALTER TABLE picking_sessions ADD COLUMN locked_since TIMESTAMPTZ;
ALTER TABLE packing_sessions ADD COLUMN locked_since TIMESTAMPTZ;
UPDATE picking_sessions SET locked_since = locked_at WHERE locked_by IS NOT NULL;
UPDATE packing_sessions SET locked_since = locked_at WHERE locked_by IS NOT NULL;

-- 'abandoned' closes a picking session packing took over (picking is
-- optional; see StartOrResumePackingUseCase).
ALTER TABLE picking_sessions DROP CONSTRAINT picking_sessions_status_check;
ALTER TABLE picking_sessions ADD CONSTRAINT picking_sessions_status_check
    CHECK (status IN ('in_progress', 'paused', 'completed', 'abandoned'));

-- +goose Down
UPDATE picking_sessions SET status = 'completed' WHERE status = 'abandoned';
ALTER TABLE picking_sessions DROP CONSTRAINT picking_sessions_status_check;
ALTER TABLE picking_sessions ADD CONSTRAINT picking_sessions_status_check
    CHECK (status IN ('in_progress', 'paused', 'completed'));
ALTER TABLE packing_sessions DROP COLUMN locked_since;
ALTER TABLE picking_sessions DROP COLUMN locked_since;
