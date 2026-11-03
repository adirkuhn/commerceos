package settings

// FallbackCurrency is the currency of a tenant that never picked one --
// the same default purchase orders get (purchase_orders.currency_code).
const FallbackCurrency = "USD"

// Currency is the tenant's currency: DefaultCurrency, or FallbackCurrency
// when it was never set.
func (s TenantSettings) Currency() string {
	if s.DefaultCurrency == "" {
		return FallbackCurrency
	}
	return s.DefaultCurrency
}
