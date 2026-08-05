package channels

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// MapWooStatus maps a WooCommerce product status to our own. publish is the
// only unambiguous "live" state; pending/draft are still being worked on
// (draft); private/trash are intentionally hidden (archived). Anything
// unrecognized defaults to draft rather than active, so an import never
// silently makes something unexpectedly live.
func MapWooStatus(wooStatus string) pmsDomain.ProductStatus {
	switch wooStatus {
	case "publish":
		return pmsDomain.ProductStatusActive
	case "private", "trash":
		return pmsDomain.ProductStatusArchived
	default:
		return pmsDomain.ProductStatusDraft
	}
}

// ShouldImportWooProductType reports whether this phase's importer handles
// a given WooCommerce product type. Only "simple" products map cleanly onto
// today's Product->Variant->SKU model; variable products need per-variation
// API calls and grouped/external products don't have their own price/stock
// at all — deliberately out of scope rather than imported half-correctly.
func ShouldImportWooProductType(wooType string) bool {
	return wooType == "simple"
}

// ParseWooPriceToMinorUnits converts a WooCommerce decimal price string
// (e.g. "1.49") to integer minor units (149). An empty string means "no
// price set" and maps to 0, not an error.
//
// Deliberately does not go through strconv.ParseFloat/float64: a decimal
// like "1.005" isn't exactly representable in binary floating point (it's
// actually stored as ~1.00499999999999989...), which rounds to the wrong
// cent for currency. Parsed as an exact decimal string instead.
func ParseWooPriceToMinorUnits(price string) (int64, error) {
	if price == "" {
		return 0, nil
	}
	neg := strings.HasPrefix(price, "-")
	s := strings.TrimPrefix(price, "-")

	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	wholeUnits, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parseWooPriceToMinorUnits: invalid price %q", price)
	}

	for _, r := range frac {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("parseWooPriceToMinorUnits: invalid price %q", price)
		}
	}
	for len(frac) < 3 {
		frac += "0"
	}
	cents, _ := strconv.ParseInt(frac[:2], 10, 64)
	if frac[2] >= '5' {
		cents++
	}

	total := wholeUnits*100 + cents
	if neg {
		total = -total
	}
	return total, nil
}

// MapCommerceStatusToWoo is the push-direction inverse of MapWooStatus.
// Not a perfect round-trip (Woo's pending/trash both collapse from draft/
// archived on import) — pushing a CommerceOS product always produces one of
// Woo's three primary states.
func MapCommerceStatusToWoo(status pmsDomain.ProductStatus) string {
	switch status {
	case pmsDomain.ProductStatusActive:
		return "publish"
	case pmsDomain.ProductStatusArchived:
		return "private"
	default:
		return "draft"
	}
}

// wooWeightToGrams/wooLengthToMM are the store's configured unit -> our
// fixed unit, per WooCommerce's own settings option values
// (woocommerce_weight_unit: kg/g/lbs/oz; woocommerce_dimension_unit:
// m/cm/mm/in/yd).
var wooWeightToGrams = map[string]float64{
	"kg": 1000, "g": 1, "lbs": 453.59237, "oz": 28.349523125,
}
var wooLengthToMM = map[string]float64{
	"m": 1000, "cm": 10, "mm": 1, "in": 25.4, "yd": 914.4,
}

// convertWooWeight converts a WooCommerce weight string (in the store's
// configured unit) to whole grams. Returns nil if the value is empty or the
// unit isn't one WooCommerce actually uses — never guesses a unit, since
// importing "0.5" as 0.5g instead of 0.5kg would silently import a value
// 1000x too small.
func convertWooWeight(value, unit string) *int64 {
	return convertWooMeasurement(value, unit, wooWeightToGrams)
}

// convertWooLength converts a WooCommerce length/width/height string (in
// the store's configured unit) to whole millimeters. Same "never guess a
// unit" reasoning as convertWooWeight.
func convertWooLength(value, unit string) *int64 {
	return convertWooMeasurement(value, unit, wooLengthToMM)
}

func convertWooMeasurement(value, unit string, unitToBase map[string]float64) *int64 {
	if value == "" {
		return nil
	}
	factor, ok := unitToBase[unit]
	if !ok {
		return nil
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	grams := int64(f*factor + 0.5) // round to nearest, f/factor always >= 0 for a real product
	return &grams
}

// gramsToWooWeight/mmToWooLength are the export-direction inverse of
// convertWooWeight/convertWooLength — our fixed unit -> the store's
// configured unit, as a decimal string Woo's API accepts. Same "never guess
// a unit" reasoning: an unrecognized/unknown store unit returns ok=false
// rather than silently picking one.
func gramsToWooWeight(grams int64, unit string) (string, bool) {
	return convertToWooMeasurement(grams, unit, wooWeightToGrams)
}

func mmToWooLength(mm int64, unit string) (string, bool) {
	return convertToWooMeasurement(mm, unit, wooLengthToMM)
}

func convertToWooMeasurement(base int64, unit string, unitToBase map[string]float64) (string, bool) {
	factor, ok := unitToBase[unit]
	if !ok {
		return "", false
	}
	return strconv.FormatFloat(float64(base)/factor, 'f', -1, 64), true
}

// FormatMinorUnitsAsWooPrice is the push-direction inverse of
// ParseWooPriceToMinorUnits — formats minor units (149) as the decimal
// string WooCommerce expects ("1.49"). Also avoids float64: dividing by 100
// in floating point can reintroduce the same representability issue
// ParseWooPriceToMinorUnits was written to avoid.
func FormatMinorUnitsAsWooPrice(minorUnits int64) string {
	neg := minorUnits < 0
	if neg {
		minorUnits = -minorUnits
	}
	whole := minorUnits / 100
	cents := minorUnits % 100
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%02d", sign, whole, cents)
}

// WooSaleFields formats a SKU's SalePriceMinorUnits/SaleEndDate as the
// strings Woo's sale_price/date_on_sale_to fields expect -- both empty
// when there's no active sale, which is the shape every caller building a
// WooProduct needs (push, sync-on-change).
func WooSaleFields(sku pmsDomain.SKU) (salePrice, saleEndDate string) {
	if sku.SalePriceMinorUnits == nil {
		return "", ""
	}
	salePrice = FormatMinorUnitsAsWooPrice(*sku.SalePriceMinorUnits)
	if sku.SaleEndDate != nil {
		saleEndDate = sku.SaleEndDate.Format(time.DateOnly)
	}
	return salePrice, saleEndDate
}
