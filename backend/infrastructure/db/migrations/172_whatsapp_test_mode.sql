-- +goose Up

-- ProviderWhatsApp-only columns (NULL/false for every other provider).
-- When enabled, every outbound WhatsApp send from this connection
-- (inbound-message replies, order-status/shipment-delivery notifications,
-- Message Center automation) is redirected to whatsapp_test_phone_e164
-- instead of the real customer -- see application/whatsapp's
-- buildEvolutionClient. Lets a merchant test the integration end-to-end
-- without risking a message reaching a real customer.
ALTER TABLE integration_connections ADD COLUMN whatsapp_test_mode_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE integration_connections ADD COLUMN whatsapp_test_phone_e164 TEXT;

-- +goose Down
ALTER TABLE integration_connections DROP COLUMN whatsapp_test_mode_enabled;
ALTER TABLE integration_connections DROP COLUMN whatsapp_test_phone_e164;
