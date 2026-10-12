package inventory

import (
	"context"
	"time"
)

// CloseToExpiryBatch is one batch with stock left that expires within a
// close-to-expiry window -- a read model for the close-to-expiry screen,
// with what someone needs to decide whether to put it on promotion.
type CloseToExpiryBatch struct {
	BatchID       string
	LotNumber     string
	ExpiryDate    time.Time
	Quantity      int64
	WarehouseID   string
	WarehouseName string
	SKUID         string
	SKUCode       string
	ProductID     string
	ProductName   string
	// ImageURL is the variant's featured image, else the product's first
	// image; "" when it has none.
	ImageURL            string
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	SaleEndDate         *time.Time
	CurrencyCode        string
	// AcknowledgedAt is set when the batch was marked as seen for the
	// window it was listed for.
	AcknowledgedAt *time.Time
	// Clearance is the promotion still selling this batch off, if any.
	Clearance *ClearancePromotionRef
}

// ActiveSalePrice is the SKU's sale price when a sale runs today (through
// its end date, inclusive), else nil.
func (b CloseToExpiryBatch) ActiveSalePrice(today time.Time) *int64 {
	return activeSalePrice(b.SalePriceMinorUnits, b.SaleEndDate, today)
}

// ClearancePromotionRef names the promotion running a batch's clearance.
type ClearancePromotionRef struct {
	PromotionID   string
	PromotionName string
}

// CloseToExpiryQuery selects one window's batches: expiring From..To
// (calendar dates, inclusive), either the ones still to look at or the
// ones already acknowledged for WindowDays.
type CloseToExpiryQuery struct {
	From, To     time.Time
	WindowDays   int
	Acknowledged bool
	Limit        int
	Offset       int
}

// CloseToExpiryCounts is how many batches a window holds, split by whether
// they were acknowledged for it.
type CloseToExpiryCounts struct {
	Open         int
	Acknowledged int
}

// CloseToExpiryReader lists sellable stock (quantity above zero, batch not
// flagged expired) by batch, sorted by expiry date, then product name.
type CloseToExpiryReader interface {
	ListCloseToExpiryBatches(ctx context.Context, q CloseToExpiryQuery) ([]CloseToExpiryBatch, error)
	CountCloseToExpiryBatches(ctx context.Context, from, to time.Time, windowDays int) (CloseToExpiryCounts, error)
}
