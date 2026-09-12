-- +goose Up
-- Supports GS1 "prefix 2x" variable-weight EAN-13 barcodes (butcher/deli
-- scale labels): every printed package of a given product has a different
-- full barcode since the actual weight is baked into it, so it can never
-- be matched by skus.barcode/barcode_aliases exact lookup. scale_code is
-- the 5-digit code the tenant programs into their scale for this SKU --
-- ResolveBarcodeUseCase decodes a scanned code (see
-- domain/pms.DecodeWeightEmbeddedBarcode) and looks it up here instead.
ALTER TABLE skus ADD COLUMN scale_code TEXT;
CREATE UNIQUE INDEX skus_tenant_scale_code_idx ON skus (tenant_id, scale_code) WHERE scale_code IS NOT NULL AND scale_code <> '';

-- pricing_mode governs how the sale price is computed once a
-- weight-embedded barcode resolves to this SKU: 'fixed' (default) means
-- every package sells for price_minor_units regardless of its actual
-- weight (e.g. calabresa, kit feijoada -- same price per package,
-- different weights); 'per_weight' means the total is
-- price_per_kilo_minor_units times the scanned package's actual weight
-- (e.g. dry meat). Independent of unit_type, which governs how *stock* is
-- denominated (kilo = grams, unit/box = count).
ALTER TABLE skus ADD COLUMN pricing_mode TEXT NOT NULL DEFAULT 'fixed';
ALTER TABLE skus ADD CONSTRAINT skus_pricing_mode_check CHECK (pricing_mode IN ('fixed', 'per_weight'));

ALTER TABLE skus ADD COLUMN price_per_kilo_minor_units BIGINT;
ALTER TABLE skus ADD CONSTRAINT skus_price_per_kilo_required_check CHECK (
    (pricing_mode = 'per_weight' AND price_per_kilo_minor_units IS NOT NULL AND price_per_kilo_minor_units > 0)
    OR (pricing_mode = 'fixed' AND price_per_kilo_minor_units IS NULL)
);

-- +goose Down
ALTER TABLE skus DROP CONSTRAINT skus_price_per_kilo_required_check;
ALTER TABLE skus DROP COLUMN price_per_kilo_minor_units;
ALTER TABLE skus DROP CONSTRAINT skus_pricing_mode_check;
ALTER TABLE skus DROP COLUMN pricing_mode;
DROP INDEX IF EXISTS skus_tenant_scale_code_idx;
ALTER TABLE skus DROP COLUMN scale_code;
