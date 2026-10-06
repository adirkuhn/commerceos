package planning

import (
	"context"
	"time"
)

// CatalogReader lists every non-virtual SKU with its stock on hand.
type CatalogReader interface {
	ListPlanningSKUs(ctx context.Context) ([]PlanningSKU, error)
}

// DemandReader reads net units sold (refunds excluded, same order
// statuses as revenue reports) per SKU, in stock units.
type DemandReader interface {
	UnitsSoldPerSKUBetween(ctx context.Context, from, to time.Time) (map[string]int64, error)
	FirstSalePerSKU(ctx context.Context) (map[string]time.Time, error)
}

// OpenOrderQuantity is stock already on purchase orders but not received,
// in stock units.
type OpenOrderQuantity struct {
	// Sent: placed with the supplier.
	Sent int64
	// Draft: on a draft / needs-review PO -- not placed yet, but counted so
	// a planned order isn't recommended twice.
	Draft int64
}

func (q OpenOrderQuantity) Total() int64 { return q.Sent + q.Draft }

// Sourcing is where a SKU was last bought: the newest non-cancelled
// purchase order line for it that has a supplier.
type Sourcing struct {
	SupplierID   string
	SupplierName string
	// LeadTimeDays is nil when the supplier has none set.
	LeadTimeDays          *int
	UnitCostMinorUnits    int64
	CurrencyCode          string
	UnitsPerPurchasedItem int64
	LastOrderedAt         time.Time
}

type SupplyReader interface {
	OpenOrderQuantities(ctx context.Context) (map[string]OpenOrderQuantity, error)
	LatestSourcing(ctx context.Context) (map[string]Sourcing, error)
}
