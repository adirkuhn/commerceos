-- +goose Up
-- tracks_weight marks a SKU whose stock quantity is denominated in grams
-- instead of whole units -- e.g. a 20kg lot of dried meat booked as one
-- stock-in of 20000, before it's portioned into individually-weighed
-- sellable items. Purely advisory for the UI (kg input instead of a plain
-- count) and for PortionFromBulkUseCase's guard that its source SKU
-- actually is a weight pool -- the stock_movements ledger itself already
-- stores a plain int64 quantity, so no ledger schema change is needed.
ALTER TABLE skus ADD COLUMN tracks_weight BOOLEAN NOT NULL DEFAULT false;

-- portioned_from_sku_id is nil unless this SKU is one sellable item peeled
-- off a bulk (tracks_weight) SKU's weight pool -- e.g. a "378g pack" SKU
-- portioned from a "Carne Seca — bulk" SKU. Unlike case_of_sku_id (a fixed
-- 1:1 N-multiplier relationship, see 043_sku_case_pack.sql), many different
-- portioned_from_sku_id rows can point at the same bulk SKU, each
-- consuming its own actual weight rather than a fixed ratio (see
-- application/inventory.PortionFromBulkUseCase). Validated in
-- application/pms's CreateSKUUseCase/UpdateSKUUseCase: the referenced SKU
-- must have tracks_weight=true, and a tracks_weight SKU can't itself be
-- portioned from another one (depth-limited to one level, same reasoning
-- as ErrCaseOfSKUCannotChain).
ALTER TABLE skus ADD COLUMN portioned_from_sku_id UUID NULL REFERENCES skus(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE skus DROP COLUMN portioned_from_sku_id;
ALTER TABLE skus DROP COLUMN tracks_weight;
