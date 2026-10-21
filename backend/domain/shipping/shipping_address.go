package shipping

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/adirkuhn/commerceos/domain/errs"
)

// ShippingAddress is the tenant's own address as carriers print it on a
// label: the sender, or where undeliverable parcels go back to (see
// ShippingPreferences).
type ShippingAddress struct {
	Company      string
	Street       string
	Number       string
	NumberSuffix string
	PostalCode   string
	City         string
	CC           string // ISO 3166-1 alpha-2
	Email        string
	Phone        string
}

var (
	ErrShippingAddressInvalid = errs.New("SHIPPING_ADDRESS_INVALID", "a shipping address needs a company name, street, house number, postal code, city and a 2-letter country code")
	countryCodePattern        = regexp.MustCompile(`^[A-Z]{2}$`)
)

// NewShippingAddress trims and normalizes in and checks it has everything
// a label needs.
func NewShippingAddress(in ShippingAddress) (ShippingAddress, error) {
	a := ShippingAddress{
		Company: strings.TrimSpace(in.Company), Street: strings.TrimSpace(in.Street),
		Number: strings.TrimSpace(in.Number), NumberSuffix: strings.TrimSpace(in.NumberSuffix),
		PostalCode: strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(in.PostalCode), " ", "")),
		City:       strings.TrimSpace(in.City), CC: strings.ToUpper(strings.TrimSpace(in.CC)),
		Email: strings.TrimSpace(in.Email), Phone: strings.TrimSpace(in.Phone),
	}
	if a.Company == "" || a.Street == "" || a.Number == "" || a.PostalCode == "" || a.City == "" || !countryCodePattern.MatchString(a.CC) {
		return ShippingAddress{}, ErrShippingAddressInvalid
	}
	if a.Email != "" {
		if parsed, err := mail.ParseAddress(a.Email); err != nil || parsed.Address != a.Email {
			return ShippingAddress{}, ErrShippingAddressInvalid
		}
	}
	return a, nil
}
