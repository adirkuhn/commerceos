package events

// Exchange-rate actions (backend/application/fx). A tenant has one live
// rate per currency, so entity_id is the currency code.
var (
	ExchangeRateSet = defineAction("fx.exchange_rate.set",
		"A currency's exchange rate was set (changes: rate_basis_points; before is null for a new currency).")
	ExchangeRateDeleted = defineAction("fx.exchange_rate.deleted",
		"A currency's exchange rate was removed. data: rate_basis_points (the removed rate, null if none was set).")
)
