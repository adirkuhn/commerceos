-- +goose Up
-- The company language is now one of the languages CommerceOS speaks
-- (domain/locale). Earlier the admin offered de/fr and saved anything; those
-- and never-set values become English, the same default the app applied.
UPDATE tenant_settings SET language = 'en' WHERE language NOT IN ('en', 'pt-BR', 'nl');
ALTER TABLE tenant_settings ALTER COLUMN language SET DEFAULT 'en';
ALTER TABLE tenant_settings ADD CONSTRAINT tenant_settings_language_check CHECK (language IN ('en', 'pt-BR', 'nl'));

-- +goose Down
ALTER TABLE tenant_settings DROP CONSTRAINT tenant_settings_language_check;
ALTER TABLE tenant_settings ALTER COLUMN language SET DEFAULT '';
