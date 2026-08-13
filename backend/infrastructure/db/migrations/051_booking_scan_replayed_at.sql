-- +goose Up
-- Marks a BookingScan row as already applied to real inventory --
-- CompleteBookingUseCase's replay skips any row with replayed_at set, so
-- retrying Finish on a booking that already partially replayed (e.g. it
-- crashed between replaying scans and marking the booking completed) never
-- re-applies a receive/correction/damaged/lost row a second time.
ALTER TABLE booking_scans ADD COLUMN replayed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE booking_scans DROP COLUMN replayed_at;
