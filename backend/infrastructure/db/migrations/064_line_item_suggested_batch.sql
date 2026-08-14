-- +goose Up
-- Suggested lot number/expiry date parsed from a supplier document (common
-- on invoices, next to each line item -- see extractLineItemsPrompt).
-- Informational only, same "suggest, don't auto-apply" pattern as
-- purchase_orders.suggested_supplier_name: nothing ever writes to a real
-- inventory batch from these columns directly. The scanner app's booking
-- screen pre-fills its lot number/expiry inputs from them, and a human
-- still has to confirm at booking time (see CompleteBookingUseCase).
ALTER TABLE purchase_order_line_items ADD COLUMN suggested_lot_number TEXT NOT NULL DEFAULT '';
ALTER TABLE purchase_order_line_items ADD COLUMN suggested_expiry_date DATE;

-- +goose Down
ALTER TABLE purchase_order_line_items DROP COLUMN suggested_expiry_date;
ALTER TABLE purchase_order_line_items DROP COLUMN suggested_lot_number;
