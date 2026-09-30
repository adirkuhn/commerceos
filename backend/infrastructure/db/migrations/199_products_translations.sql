-- +goose Up
-- A product's name/description in the store's other languages:
-- {"<lang>": {"name": "...", "description": "..."}}. See
-- pms.ValidProductTranslations. Shopper search also matches translated names.
ALTER TABLE products ADD COLUMN translations JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE products DROP COLUMN translations;
