-- +goose Up

ALTER TABLE integration_connections ADD COLUMN default_warehouse_id UUID NULL REFERENCES warehouses(id);

-- +goose Down

ALTER TABLE integration_connections DROP COLUMN default_warehouse_id;
