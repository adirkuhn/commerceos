-- +goose Up
-- The language of the merchant who opened an edit session: the theme
-- editor (on the storefront's origin) speaks it without the language
-- travelling in the URL. See domain/locale.
ALTER TABLE theme_sessions
    ADD COLUMN locale VARCHAR(8) NOT NULL DEFAULT 'en',
    ADD CONSTRAINT theme_sessions_locale_check CHECK (locale IN ('en', 'pt-BR', 'nl'));

-- +goose Down
ALTER TABLE theme_sessions
    DROP CONSTRAINT theme_sessions_locale_check,
    DROP COLUMN locale;
