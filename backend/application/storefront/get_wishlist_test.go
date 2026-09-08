package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestGetWishlist_returnsAnExistingWishlist(t *testing.T) {
	wishlistRepo := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", ProductIDs: []string{"product-1"}},
	}}
	uc := appstorefront.NewGetWishlistUseCase(newMockGormDB(t, 1), wishlistRepo)

	wishlist, err := uc.Execute(context.Background(), "token-1")

	require.NoError(t, err)
	require.Equal(t, []string{"product-1"}, wishlist.ProductIDs)
}

func TestGetWishlist_notFound(t *testing.T) {
	uc := appstorefront.NewGetWishlistUseCase(newMockGormDB(t, 1), &fakeWishlistRepo{})

	_, err := uc.Execute(context.Background(), "missing-token")

	require.ErrorIs(t, err, storefrontDomain.ErrWishlistNotFound)
}
