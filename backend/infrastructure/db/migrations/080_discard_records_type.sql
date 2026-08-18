-- +goose Up

-- Every discard_records row so far came from DiscardBatchUseCase, which
-- always records a Lost movement (expired/general write-off) -- backfill
-- existing rows as 'lost'. Going forward, MarkDamagedOrLostUseCase writes
-- this row itself for EVERY caller (the inventory page's Damaged/Lost
-- button, a booking's damaged/lost scan replay, and expiry write-off), so
-- the type the user actually picked (damaged vs lost) is preserved instead
-- of being collapsed to a fixed value -- this is what lets the unified
-- discarded/damaged/lost list distinguish them.
ALTER TABLE discard_records ADD COLUMN type TEXT NOT NULL DEFAULT 'lost';
ALTER TABLE discard_records ALTER COLUMN type DROP DEFAULT;

-- +goose Down
ALTER TABLE discard_records DROP COLUMN type;
