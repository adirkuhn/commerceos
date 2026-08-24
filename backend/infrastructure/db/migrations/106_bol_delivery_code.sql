-- +goose Up

-- Bol's Fulfilment.deliveryCode is required on every FBR offer -- NULL
-- means PushProductToBolUseCase falls back to the app-level default
-- ("3-5d", see domain/integrations.DefaultBolDeliveryCode). NULL for every
-- non-Bol connection, same nullability reasoning as bol_economic_operator_id
-- (095_bol_integration.sql).
ALTER TABLE integration_connections ADD COLUMN bol_delivery_code TEXT;

-- +goose Down
ALTER TABLE integration_connections DROP COLUMN bol_delivery_code;
