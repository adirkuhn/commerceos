-- +goose Up
-- Discounts every SKU's effective price when shown/sold through this
-- channel -- basis points off list price (e.g. 2000 = 20% off), NULL = no
-- adjustment. Meaningful only for ChannelTypeInternal today -- what lets a
-- tenant run a B2B storefront at wholesale prices alongside a B2C one at
-- retail with no customer-group/login concept (see application/storefront).
ALTER TABLE sales_channels ADD COLUMN price_adjustment_basis_points INT
    CHECK (price_adjustment_basis_points IS NULL OR (price_adjustment_basis_points >= 0 AND price_adjustment_basis_points <= 10000));

-- +goose Down
ALTER TABLE sales_channels DROP COLUMN price_adjustment_basis_points;
