package storefront_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/storefront"
)

func TestCart_IsCheckedOut(t *testing.T) {
	require.False(t, storefront.Cart{}.IsCheckedOut())
	now := time.Now()
	require.True(t, storefront.Cart{CheckedOutAt: &now}.IsCheckedOut())
}

func TestCart_SubtotalMinorUnits(t *testing.T) {
	require.Equal(t, int64(0), storefront.Cart{}.SubtotalMinorUnits())

	cart := storefront.Cart{Items: []storefront.CartItem{
		{Quantity: 2, PriceMinorUnits: 500},
		{Quantity: 1, PriceMinorUnits: 300},
	}}
	require.Equal(t, int64(1300), cart.SubtotalMinorUnits())
}

func TestCart_CurrencyCode(t *testing.T) {
	require.Equal(t, "", storefront.Cart{}.CurrencyCode())

	cart := storefront.Cart{Items: []storefront.CartItem{
		{CurrencyCode: "USD"},
		{CurrencyCode: "EUR"},
	}}
	require.Equal(t, "USD", cart.CurrencyCode())
}
