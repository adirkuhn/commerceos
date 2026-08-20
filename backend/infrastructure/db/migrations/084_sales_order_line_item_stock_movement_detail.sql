-- +goose Up
-- Denormalizes the batch/quantity a linked stock movement actually hit onto
-- the link row itself. Restocking (order cancelled/failed after decrement)
-- needs to reverse exactly what was decremented, batch for batch -- without
-- this, restock could only ever credit the general ledger back, which is
-- wrong once AllocateSaleStockUseCase can split a single line item's
-- decrement across several real batches (see that use case's doc). Storing
-- it here avoids a second cross-domain lookup back into stock_movements for
-- every restock.
ALTER TABLE sales_order_line_item_stock_movements
    ADD COLUMN batch_id UUID NULL REFERENCES batches(id) ON DELETE SET NULL,
    ADD COLUMN quantity_delta BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sales_order_line_item_stock_movements
    DROP COLUMN batch_id,
    DROP COLUMN quantity_delta;
