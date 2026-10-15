-- +goose Up
-- A weighed (per-kilo) line's total weight in grams -- NULL for a unit line.
-- A weight-pool SKU's stock is counted in grams, so its sale decrements this.
ALTER TABLE sales_order_line_items
    ADD COLUMN weight_grams BIGINT,
    ADD CONSTRAINT sales_order_line_items_weight_grams_check CHECK (weight_grams > 0);

-- +goose Down
ALTER TABLE sales_order_line_items
    DROP CONSTRAINT sales_order_line_items_weight_grams_check,
    DROP COLUMN weight_grams;
