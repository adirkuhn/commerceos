-- +goose Up
-- Additional per-line-quantity discount tiers on top of
-- price_adjustment_basis_points -- "buy N+ of one SKU, get this bigger
-- discount" (see domain/channels.SalesChannel.QuantityBreaks). Whole-
-- document JSONB, same convention as theme_pages.blocks (migration 149)
-- and storefront_carts.items (migration 146): a short list wholly owned
-- by the channel, always rewritten wholesale from the admin form.
ALTER TABLE sales_channels ADD COLUMN quantity_breaks JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE sales_channels DROP COLUMN quantity_breaks;
