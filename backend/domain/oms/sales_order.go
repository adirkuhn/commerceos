package oms

import "time"

// SalesOrderSource identifies where a completed sale was pushed in from.
// Deliberately not called "Order"/"orders" anywhere in this package --
// procurement.PurchaseOrder is inbound supply from a supplier, this is an
// outbound sale to a customer. The two must never be confused.
type SalesOrderSource string

const (
	SalesOrderSourcePOS         SalesOrderSource = "pos"
	SalesOrderSourceManual      SalesOrderSource = "manual"
	SalesOrderSourceWooCommerce SalesOrderSource = "woocommerce"
	SalesOrderSourceBol         SalesOrderSource = "bol"
	// SalesOrderSourceWhatsApp is an order placed through the WhatsApp
	// integration's chat-based checkout (see application/whatsapp) --
	// always created Pending, never already paid.
	SalesOrderSourceWhatsApp SalesOrderSource = "whatsapp"
	// SalesOrderSourceInternal is an order placed through a built-in
	// CommerceOS storefront's own checkout (see application/storefront,
	// ChannelTypeInternal) -- always created Pending, never already paid,
	// same convention as SalesOrderSourceWhatsApp.
	SalesOrderSourceInternal SalesOrderSource = "internal"
)

// SalesOrderStatus mirrors WooCommerce's order status set exactly (same
// values, same meaning) -- lets a synced/mirrored order round-trip status
// without a translation table.
type SalesOrderStatus string

const (
	SalesOrderStatusPending    SalesOrderStatus = "pending"
	SalesOrderStatusProcessing SalesOrderStatus = "processing"
	SalesOrderStatusOnHold     SalesOrderStatus = "on-hold"
	// SalesOrderStatusShipped is not one of WooCommerce's own statuses (see
	// this type's doc comment on mirroring Woo) -- added specifically for a
	// non-Woo order (pos/manual/bol) handed to a carrier via MyParcel's
	// order_status_change webhook ("processed"), since none of Woo's own
	// values mean "in transit." A Woo-sourced order uses OnHold for the same
	// transition instead, to keep its status round-trippable against Woo's
	// own vocabulary. See application/shipping.HandleOrderStatusWebhookUseCase.
	SalesOrderStatusShipped   SalesOrderStatus = "shipped"
	SalesOrderStatusCompleted SalesOrderStatus = "completed"
	SalesOrderStatusCancelled SalesOrderStatus = "cancelled"
	SalesOrderStatusRefunded  SalesOrderStatus = "refunded"
	SalesOrderStatusFailed    SalesOrderStatus = "failed"
	// SalesOrderStatusPartiallyRefunded is not one of WooCommerce's own
	// statuses (see this type's doc comment on mirroring Woo) -- set by
	// CompletePickingUseCase's replay when a PickScanActionRefund row
	// successfully refunds through the order's payment provider (see
	// SalesOrder.PaymentProvider/PaymentExternalID). Never set for an order
	// with no known provider info -- that case sets NeedsManualRefundReview
	// instead, leaving Status untouched.
	SalesOrderStatusPartiallyRefunded SalesOrderStatus = "partially_refunded"
)

// StatusReducesStock mirrors WooCommerce's own stock semantics: pending
// (payment not yet made) holds no stock impact; on-hold, processing,
// shipped, and completed all mean stock has been taken. Cancelled/failed are
// the two statuses that return stock -- refunded deliberately does not
// auto-restock here, since a refund's actual stock disposition (returned to
// sellable, damaged, lost) is a separate staff decision made through the
// stock correction flow, not something this status transition can infer.
func (s SalesOrderStatus) ReducesStock() bool {
	switch s {
	case SalesOrderStatusOnHold, SalesOrderStatusProcessing, SalesOrderStatusShipped, SalesOrderStatusCompleted:
		return true
	default:
		return false
	}
}

// Restocks reports whether transitioning *into* this status should return
// previously-decremented stock -- see ReducesStock's doc for why refunded
// is excluded.
func (s SalesOrderStatus) Restocks() bool {
	switch s {
	case SalesOrderStatusCancelled, SalesOrderStatusFailed:
		return true
	default:
		return false
	}
}

// StockStatus is a SalesOrderLineItem's decrement obligation state -- see
// the field's own doc comment.
type StockStatus string

const (
	StockStatusDecremented StockStatus = "decremented"
	StockStatusPending     StockStatus = "pending"
)

// Address mirrors WooCommerce's billing/shipping address object exactly
// (same field names/shape) -- a SalesOrder snapshots the customer's address
// as of the sale (Woo does the same: an order's billing/shipping is a copy,
// not a live reference to the Customer), which is why this is embedded
// twice on SalesOrder rather than resolved through CustomerID. Duplicated
// in domain/customers rather than imported -- domain packages never import
// each other in this codebase.
type Address struct {
	FirstName string
	LastName  string
	Company   string
	Address1  string
	Address2  string
	City      string
	State     string
	Postcode  string
	Country   string
	// Email is only meaningful on Billing -- Woo's shipping address has no
	// email field either.
	Email string
	Phone string
}

// SalesOrder is a completed sale, pushed in from an external source
// (today: a POS desktop, see application/pos and
// application/oms.ReceivePOSSalesOrderUseCase) or eventually synced from a
// channel like WooCommerce -- field set mirrors WooCommerce's order object
// so either source round-trips cleanly. Idempotent on
// (TenantID, Source, ExternalOrderID) -- the source's own retry of an
// already-received sale is a no-op, not a duplicate.
type SalesOrder struct {
	ID       string
	TenantID string
	Source   SalesOrderSource
	// ExternalOrderID is the source system's own reference for this sale --
	// what makes retries idempotent. For a Woo-sourced order this is the
	// numeric Woo order ID as a string (see application/channels' Woo
	// import) -- there's no separate typed WooOrderID column; nothing
	// ever needed the numeric type, and keeping one fact in one place
	// avoids the two silently drifting.
	ExternalOrderID string
	// ConnectionID is set on import for an order sourced from a specific
	// integration connection -- nil for POS/manual orders, and for a
	// Woo-sourced order imported before this field existed (see migration
	// 063's backfill). Lets "delete all imported products" for a
	// connection find and remove that connection's own orders first.
	ConnectionID *string
	// SalesChannelID is set only for SalesOrderSourceInternal orders --
	// which specific channels.SalesChannel the checkout happened on. Unlike
	// ConnectionID (an integration's own connection, shared across however
	// many channels use it), this exists because a tenant can run more than
	// one internal storefront (e.g. B2C and B2B) with no other way to tell
	// which one a given order came from -- WhatsApp/Woo/Bol don't need this
	// since each has at most one active channel per tenant today. See
	// application/storefront.CheckoutCartUseCase, the only writer.
	SalesChannelID *string
	// Number is the human-facing order number (Woo distinguishes this from
	// ID -- number can be customized, ID can't).
	Number   string
	OrderKey string
	Status   SalesOrderStatus

	WarehouseID string
	// CustomerID is nil for a guest/walk-in sale -- most POS sales have no
	// linked customer record.
	CustomerID   *string
	Billing      Address
	Shipping     Address
	CustomerNote string

	CurrencyCode            string
	DiscountTotalMinorUnits int64
	ShippingTotalMinorUnits int64
	TaxTotalMinorUnits      int64
	TotalMinorUnits         int64

	PaymentMethod      string
	PaymentMethodTitle string
	TransactionID      string
	// PaymentProvider/PaymentExternalID are stamped by
	// application/payments.WebhookDispatchHandler at the same moment
	// TransactionID is -- PaymentProvider is the literal provider name
	// ("stripe"/"revolut", see integrations.Provider), PaymentExternalID is
	// that provider's own checkout-session/order id. Both empty for an
	// order with no known online-payment webhook (POS, manual, most
	// imported orders) -- RefundLineItemUseCase's replay uses this pair to
	// decide whether a real refund API call is even possible; see
	// SalesOrder.NeedsManualRefundReview for what happens when it isn't.
	PaymentProvider   string
	PaymentExternalID string
	DatePaid          *time.Time
	DateCompleted     *time.Time

	// ShippingMethod/ShippingMethodTitle mirror PaymentMethod/
	// PaymentMethodTitle's shape -- ShippingMethod is the machine slug
	// (e.g. "flat_rate", "free_shipping"), ShippingMethodTitle the
	// human-facing label the customer actually saw at checkout (e.g.
	// "PostNL PakjeGemak"). Both empty means no shipping method was
	// recorded (e.g. a fully digital order, or a source that doesn't
	// report one).
	ShippingMethod      string
	ShippingMethodTitle string
	// CouponCode is the code (if any) redeemed for this order's
	// DiscountTotalMinorUnits -- empty means none, same convention as
	// ShippingMethod above.
	CouponCode string

	OccurredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// StockDecremented tracks whether stock has actually been taken for
	// this order -- see SalesOrderStatus.ReducesStock's doc. Not derived
	// from Status because a transition needs to know what already
	// happened (has it been decremented yet?) to decide what to do next.
	StockDecremented bool

	// NeedsStockReview is true when at least one line item is StockStatus
	// Pending with a SKUID set -- its decrement was attempted and failed
	// (ErrInsufficientStock), not that it was never attempted (that's
	// NeedsSKUReview). Cleared by RetryStockDecrementUseCase once every
	// such line item resolves. See the "Stock Issues" admin page.
	NeedsStockReview bool
	// NeedsSKUReview is true when at least one line item has SKUID nil --
	// an imported line whose source SKU string never matched a catalog
	// SKU. Cleared by LinkLineItemSKUUseCase once every such line item is
	// linked. See the "Unmatched SKUs" admin page.
	NeedsSKUReview bool
	// NeedsManualRefundReview is true when a PickScanActionRefund replayed
	// against this order without being able to call the payment provider's
	// refund API for real -- either because PaymentProvider/
	// PaymentExternalID are empty (no known online payment, e.g. an
	// imported order), or the API call itself failed. The line item is
	// still marked refunded either way (see SalesOrderLineItem.
	// RefundedQuantity) -- this flag just means staff must action the
	// actual money movement by hand. Never auto-cleared -- a human clears
	// it once they've handled it.
	NeedsManualRefundReview bool

	// Fulfilled is a binary secondary status, independent of Status -- true
	// once every item has physically shipped/been picked. Set automatically
	// the moment Status transitions to Completed (any path: manual edit,
	// WooCommerce import/sync) and at creation for an order that arrives
	// already completed, and explicitly by CompletePickingUseCase when a
	// picking session finishes. Never auto-cleared by a later status change
	// -- see UpdateSalesOrderInput.Fulfilled for the manual override escape
	// hatch.
	Fulfilled bool
}

// SalesOrderLineItem is one SKU sold within a SalesOrder. The POS resolves
// a barcode to a real SKU client-side before pushing the sale (see
// pms.ResolveBarcodeUseCase) for a normal catalog item, so SKUID is set
// then -- but a POS can also ring up an adhoc/custom item with no catalog
// SKU at all (ReceivePOSSalesOrderUseCase requires Name in that case,
// since there's nothing else to identify the line by), and an imported Woo
// order can carry a line item whose SKU string never matched anything in
// CommerceOS (never imported, or Woo sent none at all).
// ImportWooOrdersUseCase keeps that line rather than dropping the data,
// with SKUID nil -- same "not linked to a catalog product yet" pattern as
// procurement.PurchaseOrderLineItem.SKUID. Name/SKUCode/price/tax are
// snapshotted at sale time regardless (same as Woo's line_items.name/sku)
// so the line still reads correctly whether or not it's linked, or if the
// linked product is later renamed/deleted.
type SalesOrderLineItem struct {
	ID           string
	TenantID     string
	SalesOrderID string
	SKUID        *string
	Name         string
	SKUCode      string
	Barcode      string
	Quantity     int64
	// StockStatus tracks whether this line's stock-reducing obligation (if
	// its order's status owes one at all -- SalesOrderStatus.ReducesStock)
	// has actually been carried out. Defaults to Decremented, which also
	// covers "nothing was ever owed" (a non-stock-reducing order, or an
	// order created with SkipStockMovements) -- there is deliberately no
	// third "not applicable" value, since the order-level
	// NeedsStockReview/NeedsSKUReview rollups are what staff actually act
	// on, not this column read in isolation.
	StockStatus StockStatus

	PriceMinorUnits       int64 // per-unit price, Woo's line_items.price
	SubtotalMinorUnits    int64 // pre-discount, Woo's line_items.subtotal
	SubtotalTaxMinorUnits int64
	TotalMinorUnits       int64 // post-discount, Woo's line_items.total
	TotalTaxMinorUnits    int64
	// TaxClassID snapshots the SKU's tax classification at sale time (same
	// snapshot reasoning as Name/SKUCode) -- SubtotalTaxMinorUnits/
	// TotalTaxMinorUnits are the tax amount as computed by whoever sent the
	// order (POS, Woo, or the amount typed into a manual order); this is
	// what that amount was classified under, for audit purposes. Nil when
	// the SKU had no tax class set, or for line items created before this
	// field existed.
	TaxClassID *string

	// RefundedQuantity/RefundedAmountMinorUnits accumulate across possibly
	// more than one PickScanActionRefund row against this line (partial-by-
	// quantity, see RefundLineItemUseCase) -- RefundedAmountMinorUnits is
	// always RefundedQuantity * PriceMinorUnits at the time each refund
	// replayed, not recomputed from the line's current price. Both zero
	// means never refunded.
	RefundedQuantity         int64
	RefundedAmountMinorUnits int64

	// SoldExpiryDate is what WooCommerce reported as the expiry date sold,
	// snapshotted at import time from the commerceos-woo plugin's order-item
	// meta (commerceos_expiry_date) -- nil for every non-Woo line, and for a
	// Woo line whose product had no expiry meta set at checkout. This is
	// Woo's own belief, not necessarily the batch CommerceOS's own FEFO
	// allocation actually decremented (see SalesOrderLineItemStockMovement
	// for that) -- Woo only ever tracks one current lot/expiry per product,
	// never a real multi-batch split.
	SoldExpiryDate *time.Time

	CreatedAt time.Time

	// ProductID is display-only -- never persisted on this table, never set
	// by the repository. GetPickingSessionUseCase fills it in (via pms's
	// SKURepository/VariantRepository, same composition as procurement's
	// AttachProductDisplayNames/GetBookingUseCase) after loading a session's
	// line items -- the picking screens use it to fetch a product thumbnail.
	// Nil whenever SKUID is nil, the SKU/variant has since been deleted, or
	// the use case didn't populate it. Deliberately not a snapshot like
	// Name/SKUCode above -- a product's image can change after the sale, and
	// picking staff want the *current* one, not what it looked like at
	// order time.
	ProductID *string
}
