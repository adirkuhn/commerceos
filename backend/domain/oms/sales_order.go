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
)

// SalesOrderStatus mirrors WooCommerce's order status set exactly (same
// values, same meaning) -- lets a synced/mirrored order round-trip status
// without a translation table.
type SalesOrderStatus string

const (
	SalesOrderStatusPending    SalesOrderStatus = "pending"
	SalesOrderStatusProcessing SalesOrderStatus = "processing"
	SalesOrderStatusOnHold     SalesOrderStatus = "on-hold"
	SalesOrderStatusCompleted  SalesOrderStatus = "completed"
	SalesOrderStatusCancelled  SalesOrderStatus = "cancelled"
	SalesOrderStatusRefunded   SalesOrderStatus = "refunded"
	SalesOrderStatusFailed     SalesOrderStatus = "failed"
)

// StatusReducesStock mirrors WooCommerce's own stock semantics: pending
// (payment not yet made) holds no stock impact; on-hold, processing, and
// completed all mean stock has been taken. Cancelled/failed are the two
// statuses that return stock -- refunded deliberately does not auto-restock
// here, since a refund's actual stock disposition (returned to sellable,
// damaged, lost) is a separate staff decision made through the stock
// correction flow, not something this status transition can infer.
func (s SalesOrderStatus) ReducesStock() bool {
	switch s {
	case SalesOrderStatusOnHold, SalesOrderStatusProcessing, SalesOrderStatusCompleted:
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
	// what makes retries idempotent.
	ExternalOrderID string
	// WooOrderID is set once this order is known to WooCommerce (synced
	// from Woo, or pushed there) -- nil for a sale that only exists in
	// CommerceOS (e.g. straight from POS, never touched Woo).
	WooOrderID *int64
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
	DatePaid           *time.Time
	DateCompleted      *time.Time

	OccurredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// StockDecremented tracks whether stock has actually been taken for
	// this order -- see SalesOrderStatus.ReducesStock's doc. Not derived
	// from Status because a transition needs to know what already
	// happened (has it been decremented yet?) to decide what to do next.
	StockDecremented bool
}

// SalesOrderLineItem is one SKU sold within a SalesOrder. Unlike
// procurement.PurchaseOrderLineItem, SKUID is required (not nullable) --
// the POS resolves a barcode to a real SKU client-side before pushing the
// sale (see pms.ResolveBarcodeUseCase), so there's no "unlinked line" case
// to represent here. Name/SKUCode are snapshotted at sale time (same as
// Woo's line_items.name/sku) so the line still reads correctly even if the
// product is later renamed or the SKU deleted.
type SalesOrderLineItem struct {
	ID           string
	TenantID     string
	SalesOrderID string
	SKUID        string
	Name         string
	SKUCode      string
	Quantity     int64

	PriceMinorUnits       int64 // per-unit price, Woo's line_items.price
	SubtotalMinorUnits    int64 // pre-discount, Woo's line_items.subtotal
	SubtotalTaxMinorUnits int64
	TotalMinorUnits       int64 // post-discount, Woo's line_items.total
	TotalTaxMinorUnits    int64

	CreatedAt time.Time
}
