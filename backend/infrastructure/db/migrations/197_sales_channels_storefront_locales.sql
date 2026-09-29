-- +goose Up
-- Every language a storefront offers, its default (storefront_locale)
-- first. Shoppers switch between them; see channels.ResolveStorefrontLocales.
ALTER TABLE sales_channels
    ADD COLUMN storefront_locales JSONB NOT NULL DEFAULT '[]',
    ADD CONSTRAINT sales_channels_storefront_locales_check
        CHECK (jsonb_typeof(storefront_locales) = 'array'
               AND storefront_locales <@ '["en", "pt-BR", "nl"]'::jsonb);
UPDATE sales_channels SET storefront_locales = jsonb_build_array(storefront_locale);

-- +goose Down
ALTER TABLE sales_channels
    DROP CONSTRAINT sales_channels_storefront_locales_check,
    DROP COLUMN storefront_locales;
