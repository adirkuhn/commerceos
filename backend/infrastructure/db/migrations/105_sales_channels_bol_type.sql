-- +goose Up
-- Sales channels can now be type 'bol' -- domain/channels.ChannelTypeBol
-- has always accepted it, this CHECK constraint was simply never widened
-- when Bol support was added (same gap the OpenAPI ChannelType enum had,
-- fixed the same way in the preceding commit).
ALTER TABLE sales_channels DROP CONSTRAINT sales_channels_type_check;
ALTER TABLE sales_channels ADD CONSTRAINT sales_channels_type_check CHECK (type IN ('woocommerce', 'bol', 'internal'));

-- +goose Down
ALTER TABLE sales_channels DROP CONSTRAINT sales_channels_type_check;
ALTER TABLE sales_channels ADD CONSTRAINT sales_channels_type_check CHECK (type IN ('woocommerce', 'internal'));
