-- +goose Up
-- Comma-separated model IDs to retry in order when the primary Model hits
-- a rate limit -- see infrastructure/ai.GeminiWhatsAppIntentParser, the
-- only adapter that consumes it today (Gemini's free tier caps some
-- models at 20 requests/day, others at 500). Plain TEXT rather than a
-- Postgres array: no existing precedent for a list-typed column in this
-- codebase, and a handful of model IDs never needs indexing/querying.
ALTER TABLE ai_connections ADD COLUMN fallback_models TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE ai_connections DROP COLUMN fallback_models;
