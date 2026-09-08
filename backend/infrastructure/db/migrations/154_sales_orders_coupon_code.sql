-- +goose Up
-- Which coupon code (if any) was redeemed for this order -- empty means
-- none, same "both empty means unset" convention as
-- sales_orders.shipping_method/shipping_method_title (migration 062).
ALTER TABLE sales_orders ADD COLUMN coupon_code TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sales_orders DROP COLUMN coupon_code;
