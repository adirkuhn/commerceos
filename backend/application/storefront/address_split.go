package storefront

import (
	"regexp"
	"strings"
)

var streetAndNumberPattern = regexp.MustCompile(`^(.*?)\s+(\d+)\s*([a-zA-Z0-9-]*)$`)
var address2NumberPattern = regexp.MustCompile(`^(\d+)\s*([a-zA-Z0-9-]*)$`)

// splitStreetAndNumber is a best-effort attempt to pull a house number out
// of a checkout address for MyParcel's delivery-price-estimate call (which
// needs postcode+number, not a free-text street line) -- mirrors the admin
// app's identical frontend/src/shipping/AddressConfirmModal.tsx helper of
// the same name exactly, including its address2 fallback for the NL
// WooCommerce checkouts/plugins that split street and house number into
// separate address lines. Never trusted for anything beyond "is there a
// number to send upstream" -- estimateShipping skips the estimate
// entirely when this returns an empty number.
func splitStreetAndNumber(address1, address2 string) (street, number, suffix string) {
	trimmed1 := strings.TrimSpace(address1)
	if m := streetAndNumberPattern.FindStringSubmatch(trimmed1); m != nil {
		return m[1], m[2], m[3]
	}
	if m := address2NumberPattern.FindStringSubmatch(strings.TrimSpace(address2)); m != nil {
		return trimmed1, m[1], m[2]
	}
	return trimmed1, "", ""
}
