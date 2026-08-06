-- +goose Up
-- RawSKU preserves whatever SKU/barcode text a supplier document printed for
-- a line item, independent of RawDescription -- needed so the review UI can
-- write an accurate SupplierProductAlias (keyed on this exact raw text) when
-- a human manually links an unmatched item, so the next document from this
-- supplier auto-links instead of falling into needs_review again.
ALTER TABLE purchase_order_line_items ADD COLUMN raw_sku TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE purchase_order_line_items DROP COLUMN raw_sku;
