-- +goose Up

-- Adds 'refund' to pick_scans.action_type -- see PickScanActionRefund's doc
-- comment (domain/oms/picking.go). A refund row stages "give back part or
-- all of this line item's money", staged like every other picking action
-- until CompletePickingUseCase replays it.
ALTER TABLE pick_scans DROP CONSTRAINT pick_scans_action_type_check;
ALTER TABLE pick_scans ADD CONSTRAINT pick_scans_action_type_check
    CHECK (action_type IN ('pick', 'substitute', 'freebie', 'new_item', 'refund'));

-- +goose Down
ALTER TABLE pick_scans DROP CONSTRAINT pick_scans_action_type_check;
ALTER TABLE pick_scans ADD CONSTRAINT pick_scans_action_type_check
    CHECK (action_type IN ('pick', 'substitute', 'freebie', 'new_item'));
