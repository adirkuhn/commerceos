package inventory

import (
	"context"
	"time"
)

// ExpiringStockLine is one SKU's sellable stock sharing one expiry date --
// summed across its batches and warehouses -- with what a stock report shows
// about the product. A read model for the stock-close-to-expiry report.
type ExpiringStockLine struct {
	SKUID       string
	SKUCode     string
	ProductName string
	// ImageURL is the variant's featured image, else the product's first
	// image; "" when it has none.
	ImageURL            string
	ExpiryDate          time.Time
	Quantity            int64
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	SaleEndDate         *time.Time
	CurrencyCode        string
}

// DaysLeft is the number of days from today until the expiry date (0 =
// expires today). Both are calendar dates.
func (l ExpiringStockLine) DaysLeft(today time.Time) int {
	return DaysUntil(l.ExpiryDate, today)
}

// DaysUntil is the number of calendar days from today until expiry (0 =
// expires today, negative = already past).
func DaysUntil(expiry, today time.Time) int {
	return int(dateOnly(expiry).Sub(dateOnly(today)).Hours() / 24)
}

// ActiveSalePrice is the sale price when a sale runs today (through its end
// date, inclusive), else nil.
func (l ExpiringStockLine) ActiveSalePrice(today time.Time) *int64 {
	return activeSalePrice(l.SalePriceMinorUnits, l.SaleEndDate, today)
}

func activeSalePrice(salePrice *int64, saleEnd *time.Time, today time.Time) *int64 {
	if salePrice == nil {
		return nil
	}
	if saleEnd != nil && dateOnly(*saleEnd).Before(dateOnly(today)) {
		return nil
	}
	return salePrice
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ExpiringStockReader lists the tenant's sellable stock expiring between
// from and to (calendar dates, inclusive): quantity above zero, batch not
// flagged expired, batch not acknowledged for the windowDays window (see
// ExpiryAcknowledgement). Sorted by expiry date, then product name.
type ExpiringStockReader interface {
	ListExpiringStock(ctx context.Context, from, to time.Time, windowDays int) ([]ExpiringStockLine, error)
}
