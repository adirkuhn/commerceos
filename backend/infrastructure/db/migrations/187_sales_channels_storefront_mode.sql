-- +goose Up
-- headless: REST-only shopper API (the behaviour every channel had so far).
-- theme: CommerceOS also hosts a storefront rendered from the channel's
-- live theme. See domain/channels.StorefrontMode.
ALTER TABLE sales_channels
    ADD COLUMN storefront_mode TEXT NOT NULL DEFAULT 'headless',
    ADD CONSTRAINT sales_channels_storefront_mode_check
        CHECK (storefront_mode IN ('headless', 'theme')),
    ADD CONSTRAINT sales_channels_theme_requires_internal_check
        CHECK (storefront_mode = 'headless' OR type = 'internal');

-- +goose Down
ALTER TABLE sales_channels
    DROP CONSTRAINT sales_channels_theme_requires_internal_check,
    DROP CONSTRAINT sales_channels_storefront_mode_check,
    DROP COLUMN storefront_mode;
