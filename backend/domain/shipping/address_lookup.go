package shipping

import (
	"context"
	"github.com/adirkuhn/commerceos/domain/errs"
)

// ErrAddressNotFound is returned when no address matches the given postal
// code + house number -- a normal, expected outcome (typo, new-build not
// yet in the registry), not a fault.
var ErrAddressNotFound = errs.New("SHIPPING_ADDRESS_NOT_FOUND", "shipping: no address found for that postal code and house number")

// ErrInvalidAddressLookupInput is returned when postal code or house
// number is blank -- both required for a lookup to mean anything.
var ErrInvalidAddressLookupInput = errs.New("SHIPPING_ADDRESS_LOOKUP_INCOMPLETE", "shipping: postal code and house number are both required")

// AddressLookupResult is one confirmed street/city for a postal code +
// house number.
type AddressLookupResult struct {
	Street string
	City   string
}

// AddressLookupClient looks up a Dutch address by postal code + house
// number against the national BAG address registry -- NL-only, no API
// key, no MyParcel involvement (MyParcel has no address-validation
// endpoint reachable with a merchant API key, see AddressConfirmModal's
// own doc comment). Implemented by infrastructure/pdok.
type AddressLookupClient interface {
	// Lookup returns ErrAddressNotFound (not a generic error) when the
	// postal code and house number don't both match a single confirmed
	// address -- callers must not treat any other outcome as a match; the
	// underlying registry's free-text search always returns *some* result
	// even for a nonexistent combination, so this method's whole job is
	// verifying that result, not just relaying it.
	Lookup(ctx context.Context, postalCode, houseNumber string) (*AddressLookupResult, error)
}
