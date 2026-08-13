package oms

import "time"

// SalesOrderSource identifies where a completed sale was pushed in from.
// Deliberately not called "Order"/"orders" anywhere in this package --
// procurement.PurchaseOrder is inbound supply from a supplier, this is an
// outbound sale to a customer. The two must never be confused.
type SalesOrderSource string

const (
	SalesOrderSourcePOS SalesOrderSource = "pos"
)

// SalesOrder is a completed in-store sale, pushed in from an external
// source (today: a POS desktop, see application/pos and
// application/oms.ReceivePOSSalesOrderUseCase). Idempotent on
// (TenantID, Source, ExternalOrderID) -- the source's own retry of an
// already-received sale is a no-op, not a duplicate.
type SalesOrder struct {
	ID       string
	TenantID string
	Source   SalesOrderSource
	// ExternalOrderID is the source system's own reference for this sale --
	// what makes retries idempotent.
	ExternalOrderID string
	WarehouseID     string
	CurrencyCode    string
	TotalMinorUnits int64
	OccurredAt      time.Time
	CreatedAt       time.Time
}

// SalesOrderLineItem is one SKU sold within a SalesOrder. Unlike
// procurement.PurchaseOrderLineItem, SKUID is required (not nullable) --
// the POS resolves a barcode to a real SKU client-side before pushing the
// sale (see pms.ResolveBarcodeUseCase), so there's no "unlinked line" case
// to represent here.
type SalesOrderLineItem struct {
	ID                  string
	TenantID            string
	SalesOrderID        string
	SKUID               string
	Quantity            int64
	UnitPriceMinorUnits int64
	CreatedAt           time.Time
}
