package pms

import (
	"context"
	"time"
)

// SKUCurrencyPrice is a manual per-currency price override for one SKU --
// absent one, a SalesChannel running in a non-default currency prices this
// SKU via an fx.ExchangeRate conversion of PriceMinorUnits/CurrencyCode
// instead (see application/pms.ResolveSKUPrice, the only reader). Mirrors
// SKU's own Price/SalePrice/SaleEndDate shape so a merchant can run the
// exact same "regular + scheduled sale" pricing in a second currency, not
// just a flat converted number.
type SKUCurrencyPrice struct {
	ID                  string
	TenantID            string
	SKUID               string
	CurrencyCode        string
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	// SaleEndDate, once past, means SalePriceMinorUnits is stale --
	// checked inline by ResolveSKUPrice at read time (unlike SKU.SaleEndDate,
	// which a scheduled job clears in place) since this override drives no
	// external channel sync that would otherwise see a stale value.
	SaleEndDate *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SKUCurrencyPriceRepository interface {
	// FindBySKUAndCurrency returns nil, nil (not a typed not-found error)
	// when this SKU has no override for currencyCode -- the common case,
	// not a fault; ResolveSKUPrice falls back to fx conversion.
	FindBySKUAndCurrency(ctx context.Context, skuID, currencyCode string) (*SKUCurrencyPrice, error)
	ListBySKU(ctx context.Context, skuID string) ([]SKUCurrencyPrice, error)
	// Upsert creates or replaces the override for (SKUID, CurrencyCode) --
	// unique per pair, never a history of past overrides.
	Upsert(ctx context.Context, p *SKUCurrencyPrice) error
	Delete(ctx context.Context, skuID, currencyCode string) error
}
