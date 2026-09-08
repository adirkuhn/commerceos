-- +goose Up
-- Identity only (which coupon is attached), never a stored discount
-- amount -- domain/storefront.Cart's own doc comment on SubtotalMinorUnits
-- explains why: a Cart never caches anything derivable from its own
-- state, to avoid it silently going stale as items change. The discount
-- is always computed live from this code + the cart's current subtotal.
ALTER TABLE storefront_carts ADD COLUMN coupon_code TEXT;

-- +goose Down
ALTER TABLE storefront_carts DROP COLUMN coupon_code;
