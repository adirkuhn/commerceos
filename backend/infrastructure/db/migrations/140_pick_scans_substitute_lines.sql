-- +goose Up

-- A substitute can now be split across multiple PickScan rows -- each row
-- covers part of the original line's quantity with its own substitute SKU
-- and independently decides whether its share of the original goes back to
-- stock. See domain/oms/picking.go's doc comment for the full model.
ALTER TABLE pick_scans ADD COLUMN original_qty_covered BIGINT;
ALTER TABLE pick_scans ADD COLUMN return_to_stock BOOLEAN NOT NULL DEFAULT false;

-- Legacy substitute rows predate the split-substitute feature: the old
-- whole-line-swap use case forced quantity = original line's own quantity,
-- so quantity already IS the coverage amount for these rows.
UPDATE pick_scans SET original_qty_covered = quantity
    WHERE action_type = 'substitute' AND original_qty_covered IS NULL;

ALTER TABLE pick_scans ADD CONSTRAINT pick_scans_original_qty_covered_check
    CHECK (original_qty_covered IS NULL OR original_qty_covered > 0);
ALTER TABLE pick_scans ADD CONSTRAINT pick_scans_substitute_requires_qty_covered_check
    CHECK (action_type <> 'substitute' OR original_qty_covered IS NOT NULL);

-- +goose Down
ALTER TABLE pick_scans DROP CONSTRAINT pick_scans_substitute_requires_qty_covered_check;
ALTER TABLE pick_scans DROP CONSTRAINT pick_scans_original_qty_covered_check;
ALTER TABLE pick_scans DROP COLUMN return_to_stock;
ALTER TABLE pick_scans DROP COLUMN original_qty_covered;
