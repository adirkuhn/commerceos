-- +goose Up
-- A cancelled shipment whose carrier can't delete pre-announced labels
-- (PostNL): the label stays, unbilled unless scanned (domain/shipping.
-- CancelOutcome.LabelKept).
ALTER TABLE shipments
    ADD COLUMN cancel_label_kept BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE shipments
    DROP COLUMN IF EXISTS cancel_label_kept;
