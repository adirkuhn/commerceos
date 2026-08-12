-- +goose Up
-- Replaces the boolean tracks_weight (047_sku_bulk_portioning.sql) with an
-- explicit unit_type: "unit" (default) and "box" are both whole-count
-- stock -- box is purely descriptive, distinct from the real
-- case_of_sku_id/units_per_case multiplier relationship to another SKU.
-- "kilo" is what tracks_weight=true meant: stock quantity is grams, not a
-- count (see pms.SKU.UnitType, application/inventory.PortionFromBulkUseCase).
ALTER TABLE skus ADD COLUMN unit_type TEXT NOT NULL DEFAULT 'unit';
ALTER TABLE skus ADD CONSTRAINT skus_unit_type_check CHECK (unit_type IN ('unit', 'box', 'kilo'));

UPDATE skus SET unit_type = 'kilo' WHERE tracks_weight = true;

ALTER TABLE skus DROP COLUMN tracks_weight;

-- +goose Down
ALTER TABLE skus ADD COLUMN tracks_weight BOOLEAN NOT NULL DEFAULT false;
UPDATE skus SET tracks_weight = true WHERE unit_type = 'kilo';

ALTER TABLE skus DROP CONSTRAINT skus_unit_type_check;
ALTER TABLE skus DROP COLUMN unit_type;
