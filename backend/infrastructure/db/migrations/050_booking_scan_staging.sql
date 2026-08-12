-- +goose Up
-- BookingScan becomes a staged action (see domain/procurement/booking.go's
-- doc comment): nothing reaches real inventory or an external sales channel
-- until CompleteBookingUseCase replays these rows when the booking
-- finishes. lot_number/expiry_date let a receive row create a batch at
-- replay time (or a later row target that same not-yet-real batch, matched
-- by sku_id+lot_number); action_type distinguishes receive/correction/
-- damaged/lost; write_off_quantity holds the damaged/lost amount separately
-- from quantity, which stays the net effect on "received" progress.
ALTER TABLE booking_scans ADD COLUMN lot_number TEXT NOT NULL DEFAULT '';
ALTER TABLE booking_scans ADD COLUMN expiry_date DATE;
ALTER TABLE booking_scans ADD COLUMN action_type TEXT NOT NULL DEFAULT 'receive';
ALTER TABLE booking_scans ADD COLUMN write_off_quantity BIGINT NOT NULL DEFAULT 0;
ALTER TABLE booking_scans ADD COLUMN reason TEXT NOT NULL DEFAULT '';

ALTER TABLE booking_scans ADD CONSTRAINT booking_scans_action_type_check
    CHECK (action_type IN ('receive', 'correction', 'damaged', 'lost'));

-- +goose Down
ALTER TABLE booking_scans DROP CONSTRAINT booking_scans_action_type_check;
ALTER TABLE booking_scans DROP COLUMN reason;
ALTER TABLE booking_scans DROP COLUMN write_off_quantity;
ALTER TABLE booking_scans DROP COLUMN action_type;
ALTER TABLE booking_scans DROP COLUMN expiry_date;
ALTER TABLE booking_scans DROP COLUMN lot_number;
