-- +goose Up
-- The page a headless storefront completes shopper sign-in on. Emailed
-- sign-in links point here (hosted themed storefronts use their own
-- /login), so it is merchant configuration -- never taken from the
-- shopper's request. See domain/channels.ResolveSignInURL.
ALTER TABLE sales_channels
    ADD COLUMN sign_in_url TEXT,
    ADD CONSTRAINT sales_channels_sign_in_url_requires_internal_check
        CHECK (sign_in_url IS NULL OR type = 'internal');

-- +goose Down
ALTER TABLE sales_channels
    DROP CONSTRAINT sales_channels_sign_in_url_requires_internal_check,
    DROP COLUMN sign_in_url;
