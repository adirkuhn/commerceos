package events

// Storefront actions -- shopper-side records of the hosted storefront
// (backend/application/storefront). Shopper personal data (email, name,
// address) and secrets (cart/wishlist/session tokens, magic links) never
// go into an event: entity_id is the record's id, never its token.
var (
	CartCreated = defineAction("storefront.cart.created",
		"A shopper started a cart. data: sales_channel_id.")
	CartItemsUpdated = defineAction("storefront.cart.items_updated",
		"A cart's lines were replaced (add, remove or change quantity). data: sales_channel_id, line_count, items (sku_id, sku_code, quantity). changes: items (before/after, sku_code -> quantity).")
	CartCouponApplied = defineAction("storefront.cart.coupon_applied",
		"A coupon was applied to a cart. data: coupon_code, discount_minor_units. changes: coupon_code.")
	CartCouponRemoved = defineAction("storefront.cart.coupon_removed",
		"A cart's coupon was removed. changes: coupon_code.")
	WishlistCreated = defineAction("storefront.wishlist.created",
		"A shopper started a wishlist. data: sales_channel_id.")
	WishlistItemAdded = defineAction("storefront.wishlist.item_added",
		"A product was added to a wishlist. data: sales_channel_id, product_id.")
	WishlistItemRemoved = defineAction("storefront.wishlist.item_removed",
		"A product was removed from a wishlist. data: sales_channel_id, product_id.")
	PaymentLinkCreated = defineAction("storefront.payment_link.created",
		"A checkout payment link was created at the channel's payment provider (entity_id: the provider's link id). data: sales_channel_id, provider, reference (order number), amount_minor_units, currency_code.")
	MagicLinkRequested = defineAction("storefront.magic_link.requested",
		"A shopper asked for a sign-in link; it was emailed. data: sales_channel_id, customer_id.")
	CustomerSessionStarted = defineAction("storefront.customer_session.started",
		"A shopper signed in with a magic link (the link is now used). data: sales_channel_id, customer_id, magic_link_id.")
	CustomerSessionEnded = defineAction("storefront.customer_session.ended",
		"A shopper signed out. data: sales_channel_id, customer_id.")
	StorefrontReviewSubmitted = defineAction("storefront.review.submitted",
		"A shopper submitted a product review (awaiting moderation). data: sales_channel_id, product_id, rating, verified (bought the product).")
)
