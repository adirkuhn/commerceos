-- +goose Up
-- The language a channel's storefront speaks to shoppers (built-in texts,
-- price/date formatting). See domain/channels.StorefrontLocale.
ALTER TABLE sales_channels
    ADD COLUMN storefront_locale VARCHAR(8) NOT NULL DEFAULT 'en',
    ADD CONSTRAINT sales_channels_storefront_locale_check
        CHECK (storefront_locale IN ('en', 'pt-BR', 'nl'));

-- +goose Down
ALTER TABLE sales_channels
    DROP CONSTRAINT sales_channels_storefront_locale_check,
    DROP COLUMN storefront_locale;
