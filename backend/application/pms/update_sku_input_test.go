package pms_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// UpdateSKUInput replaces every field, so a caller changing one field must
// start from the SKU's current values -- none may be dropped.
func TestUpdateSKUInputFrom_carriesEverySKUFieldThrough(t *testing.T) {
	sale, end, w, l, wd, h, units, perKilo := int64(899), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), int64(380), int64(10), int64(20), int64(30), int64(12), int64(2999)
	caseOf, portioned, scale, tax := "sku-unit", "sku-pool", "2994721", "tax-1"
	s := pmsDomain.SKU{
		ID: "sku-1", SKUCode: "CS-380G", Barcode: "2994721003809", PriceMinorUnits: 1139, SalePriceMinorUnits: &sale, SaleEndDate: &end,
		CurrencyCode: "EUR", WeightGrams: &w, LengthMM: &l, WidthMM: &wd, HeightMM: &h, IsVirtual: true, TracksExpiry: true,
		CaseOfSKUID: &caseOf, UnitsPerCase: &units, UnitType: pmsDomain.SKUUnitTypeUnit, PortionedFromSKUID: &portioned,
		ScaleCode: &scale, PricingMode: pmsDomain.SKUPricingModePerWeight, PricePerKiloMinorUnits: &perKilo, TaxClassID: &tax,
		Status: pmsDomain.SKUStatusActive,
	}

	assert.Equal(t, apppms.UpdateSKUInput{
		ID: "sku-1", SKUCode: "CS-380G", Barcode: "2994721003809", PriceMinorUnits: 1139, SalePriceMinorUnits: &sale, SaleEndDate: &end,
		CurrencyCode: "EUR", WeightGrams: &w, LengthMM: &l, WidthMM: &wd, HeightMM: &h, IsVirtual: true, TracksExpiry: true,
		CaseOfSKUID: &caseOf, UnitsPerCase: &units, UnitType: pmsDomain.SKUUnitTypeUnit, PortionedFromSKUID: &portioned,
		ScaleCode: &scale, PricingMode: pmsDomain.SKUPricingModePerWeight, PricePerKiloMinorUnits: &perKilo, TaxClassID: &tax,
		Status: pmsDomain.SKUStatusActive,
	}, apppms.UpdateSKUInputFrom(s))
}
