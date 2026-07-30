package channels

import (
	"fmt"
	"strconv"
	"strings"

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
