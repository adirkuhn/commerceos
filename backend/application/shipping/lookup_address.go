package shipping

import (
	"context"
	"fmt"
	"strings"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// LookupAddressUseCase is AddressConfirmModal's typo/address check -- look
// up the canonical street/city for a postal code + house number, so staff
// can catch a mismatch before it ships. Stateless: no tenant/DB access at
// all, this is a public registry lookup, not anything scoped to a
// connection or tenant.
type LookupAddressUseCase struct {
	client shippingDomain.AddressLookupClient
}

func NewLookupAddressUseCase(client shippingDomain.AddressLookupClient) *LookupAddressUseCase {
	return &LookupAddressUseCase{client: client}
}

func (uc *LookupAddressUseCase) Execute(ctx context.Context, postalCode, houseNumber string) (*shippingDomain.AddressLookupResult, error) {
	postalCode = strings.TrimSpace(postalCode)
	houseNumber = strings.TrimSpace(houseNumber)
	if postalCode == "" || houseNumber == "" {
		return nil, fmt.Errorf("lookupAddress: %w", shippingDomain.ErrInvalidAddressLookupInput)
	}
	result, err := uc.client.Lookup(ctx, postalCode, houseNumber)
	if err != nil {
		return nil, fmt.Errorf("lookupAddress: %w", err)
	}
	return result, nil
}
