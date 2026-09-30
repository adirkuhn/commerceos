-- +goose Up
-- A document's title (pages) and SEO texts in the store's other languages:
-- {"<lang>": {"title": "...", "seo": {"title": "...", "description": "..."}}}.
-- See theme.ValidDocumentTranslations.
ALTER TABLE theme_documents ADD COLUMN translations JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE theme_documents DROP COLUMN translations;
