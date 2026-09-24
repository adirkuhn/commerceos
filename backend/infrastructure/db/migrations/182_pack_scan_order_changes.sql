-- +goose Up

-- Packing is where order changes happen now (picking is optional): a pack
-- scan is either a plain 'pack' (box contents) or a staged order change --
-- substitute / freebie / new_item / refund -- applied when the packing
-- session completes. Columns mirror pick_scans. A freebie/new item has no
-- line item yet (it becomes one on completion), hence line_item_id nullable.
ALTER TABLE pack_scans ALTER COLUMN line_item_id DROP NOT NULL;
ALTER TABLE pack_scans ADD COLUMN action_type TEXT NOT NULL DEFAULT 'pack'
    CONSTRAINT pack_scans_action_type_check CHECK (action_type IN ('pack', 'substitute', 'freebie', 'new_item', 'refund'));
ALTER TABLE pack_scans ADD COLUMN original_qty_covered BIGINT;
ALTER TABLE pack_scans ADD COLUMN return_to_stock BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE pack_scans ADD COLUMN price_minor_units BIGINT;
ALTER TABLE pack_scans ADD COLUMN reason TEXT NOT NULL DEFAULT '';
ALTER TABLE pack_scans ADD COLUMN replayed_at TIMESTAMPTZ;

-- +goose Down
DELETE FROM pack_scans WHERE action_type <> 'pack';
ALTER TABLE pack_scans DROP COLUMN replayed_at;
ALTER TABLE pack_scans DROP COLUMN reason;
ALTER TABLE pack_scans DROP COLUMN price_minor_units;
ALTER TABLE pack_scans DROP COLUMN return_to_stock;
ALTER TABLE pack_scans DROP COLUMN original_qty_covered;
ALTER TABLE pack_scans DROP COLUMN action_type;
ALTER TABLE pack_scans ALTER COLUMN line_item_id SET NOT NULL;
