-- +goose Up
-- A collection's texts in the store's other languages:
-- {"<lang>": {"name": "...", "description": "...", "seo_title": "...", "seo_description": "..."}}.
ALTER TABLE collections ADD COLUMN translations JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE collections DROP COLUMN translations;
