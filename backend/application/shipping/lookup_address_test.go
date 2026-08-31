package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestLookupAddress_returnsClientResult(t *testing.T) {
	client := &fakeAddressLookupClient{result: &shippingDomain.AddressLookupResult{Street: "Damrak", City: "Amsterdam"}}
	uc := appshipping.NewLookupAddressUseCase(client)

	result, err := uc.Execute(context.Background(), "1012AB", "1")

	require.NoError(t, err)
	assert.Equal(t, "Damrak", result.Street)
	assert.Equal(t, "Amsterdam", result.City)
}

func TestLookupAddress_propagatesNotFound(t *testing.T) {
	client := &fakeAddressLookupClient{err: shippingDomain.ErrAddressNotFound}
	uc := appshipping.NewLookupAddressUseCase(client)

	_, err := uc.Execute(context.Background(), "9999ZZ", "999")

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrAddressNotFound)
}

func TestLookupAddress_rejectsBlankInput(t *testing.T) {
	client := &fakeAddressLookupClient{}
	uc := appshipping.NewLookupAddressUseCase(client)

	_, err := uc.Execute(context.Background(), "", "1")
	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrInvalidAddressLookupInput)

	_, err = uc.Execute(context.Background(), "1012AB", "  ")
	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrInvalidAddressLookupInput)
}
