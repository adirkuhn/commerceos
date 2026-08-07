-- +goose Up
-- Explicit ordinal position, replacing "sort by created_at" -- a
-- parse-time batch stamps every line item on a PO with the exact same
-- created_at, so among tied rows Postgres's ORDER BY falls back to physical
-- heap order, which an UPDATE (e.g. linking a SKU) can silently change.
-- Position is set once at creation and never touched by later edits.
ALTER TABLE purchase_order_line_items ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

-- Backfill existing rows with a stable per-PO ordering. created_at/id is the
-- best available approximation of original document order for rows that
-- predate this column -- it can't recover a true original order that was
-- already lost to the bug above, only make each PO's ordering deterministic
-- going forward.
WITH ranked AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY purchase_order_id ORDER BY created_at ASC, id ASC) - 1 AS rn
    FROM purchase_order_line_items
)
UPDATE purchase_order_line_items p
SET position = ranked.rn
FROM ranked
WHERE p.id = ranked.id;

-- +goose Down
ALTER TABLE purchase_order_line_items DROP COLUMN position;
