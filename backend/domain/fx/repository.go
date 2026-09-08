package fx

import "context"

type ExchangeRateRepository interface {
	// FindByCurrency returns nil, nil (not ErrExchangeRateNotFound) when
	// the tenant hasn't configured a rate for this currency yet -- callers
	// doing price resolution (application/pms.ResolvePriceForCurrency)
	// treat "no rate" as their own typed error, since here it just means
	// "not configured", not a fault.
	FindByCurrency(ctx context.Context, currencyCode string) (*ExchangeRate, error)
	ListByTenant(ctx context.Context) ([]ExchangeRate, error)
	// Upsert creates or replaces the tenant's rate for r.CurrencyCode
	// (unique per tenant+currency) -- there is only ever one live rate per
	// currency, never a history of past rates.
	Upsert(ctx context.Context, r *ExchangeRate) error
	Delete(ctx context.Context, currencyCode string) error
}
