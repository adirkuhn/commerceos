// Package fx holds the tenant-maintained exchange rate a SalesChannel
// running in a non-default currency uses as its pricing fallback --
// SKU/coupon prices only ever get an explicit per-currency override on
// request (see pms.SKUCurrencyPrice, coupons.CouponCurrencyAmount);
// everything else is converted through here. Deliberately not a live-rate
// integration (CLAUDE.md's third-party-API approval gate) -- a human sets
// and updates these.
package fx

import (
	"github.com/adirkuhn/commerceos/domain/errs"
	"time"
)

var (
	// ErrExchangeRateNotFound is returned by a not-found FindByCurrency --
	// distinct from "conversion needed but no rate configured yet", which
	// callers doing price resolution treat as their own typed error (see
	// application/pms.ErrExchangeRateRequired) so a misconfigured channel
	// fails clearly rather than silently pricing at parity.
	ErrExchangeRateNotFound = errs.New("EXCHANGE_RATE_NOT_FOUND", "exchange rate not found")
	ErrInvalidCurrencyCode  = errs.New("EXCHANGE_RATE_INVALID_CURRENCY", "currency code must be 3 uppercase letters")
	ErrInvalidRate          = errs.New("EXCHANGE_RATE_INVALID", "exchange rate must be a positive number")
)

// ExchangeRate converts 1 minor-unit of the tenant's default currency into
// RateBasisPoints/10000 minor-units of CurrencyCode -- the same basis-point
// convention as coupons.Coupon.DiscountValue and tax.TaxRate.RateBasisPoints
// (10000 = 1:1 parity), so e.g. EUR->USD at 1.0876 is stored as 10876.
type ExchangeRate struct {
	ID              string
	TenantID        string
	CurrencyCode    string
	RateBasisPoints int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ValidCurrencyCode reports whether code is 3 uppercase ASCII letters
// (e.g. "EUR", "USD") -- shared by every domain that takes a currency code
// as user input (SalesChannel.CurrencyCode, ExchangeRate.CurrencyCode,
// per-currency price/coupon overrides) so the format check lives in
// exactly one place.
func ValidCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// Convert applies this rate to a minor-units amount in the tenant's
// default currency, rounding half away from zero -- same rounding
// CLAUDE.md's integer-money convention uses everywhere else basis points
// get applied to an amount (e.g. tax/discount calc).
func (r ExchangeRate) Convert(baseMinorUnits int64) int64 {
	if baseMinorUnits == 0 {
		return 0
	}
	negative := baseMinorUnits < 0
	if negative {
		baseMinorUnits = -baseMinorUnits
	}
	converted := (baseMinorUnits*r.RateBasisPoints + 5000) / 10000
	if negative {
		return -converted
	}
	return converted
}
