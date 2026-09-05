-- +goose Up
-- Sales channels can now be type 'whatsapp' -- domain/channels.ChannelTypeWhatsApp,
-- same widening as 105_sales_channels_bol_type.sql did for 'bol'.
ALTER TABLE sales_channels DROP CONSTRAINT sales_channels_type_check;
ALTER TABLE sales_channels ADD CONSTRAINT sales_channels_type_check CHECK (type IN ('woocommerce', 'bol', 'internal', 'whatsapp'));

-- +goose Down
ALTER TABLE sales_channels DROP CONSTRAINT sales_channels_type_check;
ALTER TABLE sales_channels ADD CONSTRAINT sales_channels_type_check CHECK (type IN ('woocommerce', 'bol', 'internal'));
