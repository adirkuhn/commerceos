package events

// SKUPricePerKiloChanged is raised when a scale-sold weight pool's price
// per kilo changes. Its packages carry their own fixed prices derived from
// that rate (see domain/pms.PackagePriceMinorUnits), so a consumer reprices
// them -- the pool itself and per-kilo scale sales use the new rate
// immediately.
type SKUPricePerKiloChanged struct {
	Base
	TenantID                  string `json:"tenant_id"`
	SKUID                     string `json:"sku_id"`
	OldPricePerKiloMinorUnits *int64 `json:"old_price_per_kilo_minor_units"`
	NewPricePerKiloMinorUnits int64  `json:"new_price_per_kilo_minor_units"`
}

func (SKUPricePerKiloChanged) EventName() string { return "pms.sku.price_per_kilo_changed" }
