package events

// Coupon actions (backend/application/coupons).
var (
	CouponCreated = defineAction("coupons.coupon.created",
		"A discount coupon was created. data: code, discount_type, discount_value, min_subtotal_minor_units, usage_limit, expires_at.")
	CouponUpdated = defineAction("coupons.coupon.updated",
		"A coupon changed (changes: code, discount_type, discount_value, min_subtotal_minor_units, usage_limit, expires_at, active).")
	CouponDeleted = defineAction("coupons.coupon.deleted",
		"A coupon was deleted.")
	CouponCurrencyAmountSet = defineAction("coupons.coupon_currency_amount.set",
		"A fixed-amount coupon's discount in one currency was set. entity_id is the coupon. data: currency_code, amount_minor_units.")
	CouponCurrencyAmountDeleted = defineAction("coupons.coupon_currency_amount.deleted",
		"A coupon's per-currency discount was removed (it falls back to FX conversion). entity_id is the coupon. data: currency_code.")
)
