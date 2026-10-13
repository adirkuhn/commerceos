package storefront

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrCartNotFound = errs.New("CART_NOT_FOUND", "cart not found")
	// ErrCartEmpty guards CheckoutCartUseCase -- an order with no line
	// items has nothing to sell, same reasoning as oms.ErrEmptyLineItems.
	ErrCartEmpty = errs.New("CART_EMPTY", "cart is empty")
	// ErrCartAlreadyCheckedOut guards CheckoutCartUseCase against a retried
	// or double-submitted checkout request re-running against a cart that
	// already produced a SalesOrder -- see Cart.CheckedOutAt.
	ErrCartAlreadyCheckedOut = errs.New("CART_ALREADY_CHECKED_OUT", "cart has already been checked out")
	// ErrProductNotAvailable covers both "no SKU with this code" and "SKU
	// exists but its product isn't enrolled in this sales channel" --
	// deliberately the same error for both, same reasoning as
	// application/whatsapp.ErrProductNotFound: a storefront request must
	// never be able to use this to probe which SKU codes exist in the
	// catalog beyond what's actually enrolled in the channel it's calling
	// through.
	ErrProductNotAvailable = errs.New("PRODUCT_NOT_AVAILABLE", "product not available on this sales channel")
	// ErrInsufficientStock guards AddCartItem-style writes against a
	// requested quantity exceeding what's currently sellable.
	ErrInsufficientStock = errs.New("INSUFFICIENT_STOCK", "insufficient stock for requested quantity")
	// ErrNoInternalChannel is returned when the resolved SalesChannel isn't
	// an active ChannelTypeInternal channel -- a storefront API key must
	// always resolve to one (see the migration adding
	// storefront_api_keys.sales_channel_id).
	ErrNoInternalChannel = errs.New("STORE_NO_INTERNAL_CHANNEL", "storefront: no active internal sales channel for this key")
	// ErrOrderNotFound covers "no order with this number", "order exists
	// but the email doesn't match", and "order exists but was placed on a
	// different sales channel" -- deliberately the same error for all
	// three (see LookupOrderUseCase), so a guest order-status lookup can
	// never be used to enumerate order numbers or guess billing emails.
	ErrOrderNotFound = errs.New("ORDER_NOT_FOUND", "order not found")
	// ErrWishlistNotFound mirrors ErrCartNotFound for Wishlist -- see
	// GetWishlistUseCase/ToggleWishlistItemUseCase.
	ErrWishlistNotFound = errs.New("WISHLIST_NOT_FOUND", "wishlist not found")
	// ErrInvalidOrExpiredMagicLink covers "no such token", "already used",
	// and "expired" alike -- deliberately the same error for all three, so
	// a captured/guessed token value can never be used to distinguish a
	// real-but-expired link from a nonexistent one.
	ErrInvalidOrExpiredMagicLink = errs.New("SIGN_IN_LINK_INVALID", "this sign-in link is invalid or has expired")
	// ErrSessionNotFound covers a missing, expired, or wrong-channel
	// session token -- same "one error for every reason" rule as
	// ErrInvalidOrExpiredMagicLink.
	ErrSessionNotFound = errs.New("SESSION_NOT_FOUND", "session not found")
	// ErrCardPaymentNotSupported guards CreatePaymentIntentUseCase/
	// CheckoutCartUseCase's embedded-card-payment path -- returned when the
	// channel's configured payment connection isn't Stripe (Revolut has no
	// equivalent client-side-confirm SDK) or Stripe's publishable key was
	// never set (see integrations.Connection.ConsumerKey's doc comment).
	ErrCardPaymentNotSupported = errs.New("CARD_PAYMENT_NOT_SUPPORTED", "embedded card payment is not available on this channel")
	// ErrCardPaymentMismatch guards checkout against confirming an order
	// against a PaymentIntent that doesn't actually match it -- the
	// server-side re-check (CardPaymentClient.RetrievePaymentIntent) came
	// back not-succeeded, or its amount/currency don't match the order
	// being placed (e.g. the cart changed after the intent was created).
	ErrCardPaymentMismatch = errs.New("CARD_PAYMENT_MISMATCH", "this payment could not be verified against the order total")
	// ErrChannelWarehouseNotConfigured guards CheckoutCartUseCase: an
	// order's stock decrement must attribute to a real warehouse
	// (sales_orders.warehouse_id is NOT NULL), so a channel with no
	// SalesChannel.DefaultWarehouseID set can't actually process one --
	// found live, not by any prior test: every internal-storefront
	// checkout failed at the DB with an invalid-uuid error before this was
	// checked explicitly.
	ErrChannelWarehouseNotConfigured = errs.New("STORE_WAREHOUSE_NOT_CONFIGURED", "this sales channel has no default warehouse configured")
	// ErrInvalidSlowMovingWindow / ErrInvalidSlowMovingLimit guard
	// ListSlowMovingProductsUseCase's bounds -- the OpenAPI request
	// validator already enforces the same ranges at the HTTP edge; these
	// keep the use case honest for any non-HTTP caller.
	ErrInvalidSlowMovingWindow = errs.New("INVALID_SLOW_MOVING_WINDOW", "days must be between 1 and 365")
	ErrInvalidSlowMovingLimit  = errs.New("INVALID_SLOW_MOVING_LIMIT", "limit must be between 1 and 50")
)
