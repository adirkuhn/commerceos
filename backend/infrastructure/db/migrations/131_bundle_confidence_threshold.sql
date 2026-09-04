-- +goose Up
ALTER TABLE tenant_settings ADD COLUMN bundle_confidence_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.30;

-- +goose Down
ALTER TABLE tenant_settings DROP COLUMN bundle_confidence_threshold;
