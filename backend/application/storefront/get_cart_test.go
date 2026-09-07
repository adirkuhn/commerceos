package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestGetCart_returnsTheCartForItsToken(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", Items: []storefrontDomain.CartItem{{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500}}},
	}}
	uc := appstorefront.NewGetCartUseCase(newMockGormDB(t, 1), cartRepo)

	cart, err := uc.Execute(context.Background(), "token-1")

	require.NoError(t, err)
	require.Equal(t, "cart-1", cart.ID)
	require.Equal(t, int64(1000), cart.SubtotalMinorUnits())
}

func TestGetCart_returnsNotFoundForUnknownToken(t *testing.T) {
	uc := appstorefront.NewGetCartUseCase(newMockGormDB(t, 1), &fakeCartRepo{})

	_, err := uc.Execute(context.Background(), "unknown-token")

	require.ErrorIs(t, err, storefrontDomain.ErrCartNotFound)
}
