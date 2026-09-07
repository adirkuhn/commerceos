package storefront

import "errors"

var (
	ErrCartNotFound = errors.New("cart not found")
	// ErrCartEmpty guards CheckoutCartUseCase -- an order with no line
	// items has nothing to sell, same reasoning as oms.ErrEmptyLineItems.
	ErrCartEmpty = errors.New("cart is empty")
	// ErrCartAlreadyCheckedOut guards CheckoutCartUseCase against a retried
	// or double-submitted checkout request re-running against a cart that
	// already produced a SalesOrder -- see Cart.CheckedOutAt.
	ErrCartAlreadyCheckedOut = errors.New("cart has already been checked out")
	// ErrProductNotAvailable covers both "no SKU with this code" and "SKU
	// exists but its product isn't enrolled in this sales channel" --
	// deliberately the same error for both, same reasoning as
	// application/whatsapp.ErrProductNotFound: a storefront request must
	// never be able to use this to probe which SKU codes exist in the
	// catalog beyond what's actually enrolled in the channel it's calling
	// through.
	ErrProductNotAvailable = errors.New("product not available on this sales channel")
	// ErrInsufficientStock guards AddCartItem-style writes against a
	// requested quantity exceeding what's currently sellable.
	ErrInsufficientStock = errors.New("insufficient stock for requested quantity")
	// ErrNoInternalChannel is returned when the resolved SalesChannel isn't
	// an active ChannelTypeInternal channel -- a storefront API key must
	// always resolve to one (see the migration adding
	// storefront_api_keys.sales_channel_id).
	ErrNoInternalChannel = errors.New("storefront: no active internal sales channel for this key")
	// ErrOrderNotFound covers "no order with this number", "order exists
	// but the email doesn't match", and "order exists but was placed on a
	// different sales channel" -- deliberately the same error for all
	// three (see LookupOrderUseCase), so a guest order-status lookup can
	// never be used to enumerate order numbers or guess billing emails.
	ErrOrderNotFound = errors.New("order not found")
)
