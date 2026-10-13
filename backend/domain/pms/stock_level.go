package pms

import "github.com/adirkuhn/commerceos/domain/errs"

// StockLevel narrows a product list by the product's total stock (every
// SKU, every warehouse). Low and out leave archived products out.
type StockLevel string

const (
	// StockLevelAny applies no stock filter.
	StockLevelAny StockLevel = ""
	// StockLevelLow: total above 0 and at or below the product's low stock
	// threshold -- its own LowStockThreshold, else the tenant's setting.
	StockLevelLow StockLevel = "low"
	// StockLevelOut: total at or below 0, including a product with no SKUs.
	StockLevelOut StockLevel = "out"
)

var (
	ErrInvalidStockLevel        = errs.New("PRODUCT_INVALID_STOCK_LEVEL", "stock filter must be low or out")
	ErrInvalidLowStockThreshold = errs.New("PRODUCT_INVALID_LOW_STOCK_THRESHOLD", "low stock threshold must be at least 0")
)

// ParseStockLevel validates a stock filter; empty means no filter.
func ParseStockLevel(s string) (StockLevel, error) {
	switch StockLevel(s) {
	case StockLevelAny, StockLevelLow, StockLevelOut:
		return StockLevel(s), nil
	default:
		return "", ErrInvalidStockLevel
	}
}

// ValidateLowStockThreshold guards Product.LowStockThreshold: nil (use the
// tenant's) or >= 0 (0 = never flag this product low).
func ValidateLowStockThreshold(threshold *int) error {
	if threshold != nil && *threshold < 0 {
		return ErrInvalidLowStockThreshold
	}
	return nil
}
