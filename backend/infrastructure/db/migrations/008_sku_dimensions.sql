-- +goose Up

ALTER TABLE skus
    ADD COLUMN weight_grams BIGINT,
    ADD COLUMN length_mm    BIGINT,
    ADD COLUMN width_mm     BIGINT,
    ADD COLUMN height_mm    BIGINT;

-- +goose Down

ALTER TABLE skus
    DROP COLUMN weight_grams,
    DROP COLUMN length_mm,
    DROP COLUMN width_mm,
    DROP COLUMN height_mm;
