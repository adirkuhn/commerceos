-- +goose Up
-- A PO no longer needs a warehouse decided up front -- it's assigned the
-- first time stock actually gets received against it (see
-- StartOrResumeBookingUseCase / the receiving review's "Receive stock"
-- action), not at order time.
ALTER TABLE purchase_orders ALTER COLUMN warehouse_id DROP NOT NULL;

-- +goose Down
-- Down is only safe if every row already has a warehouse_id -- true for any
-- database that hasn't used the nullable window, which is the only
-- situation a rollback of this migration makes sense in.
ALTER TABLE purchase_orders ALTER COLUMN warehouse_id SET NOT NULL;
