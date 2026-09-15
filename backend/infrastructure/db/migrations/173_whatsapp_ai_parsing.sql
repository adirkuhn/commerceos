-- +goose Up

-- ProviderWhatsApp-only column (false for every other provider). When
-- enabled, a message that doesn't match the rigid "COMMAND CODE" format is
-- classified by the tenant's connected AI provider (domain/whatsapp.
-- AIIntentParser) instead of falling straight to the help text -- only
-- effective when the tenant also has an AI connection configured (see
-- SetWhatsAppAIParsingUseCase, which refuses to enable this otherwise).
ALTER TABLE integration_connections ADD COLUMN whatsapp_ai_parsing_enabled BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE integration_connections DROP COLUMN whatsapp_ai_parsing_enabled;
