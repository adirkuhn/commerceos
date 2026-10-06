package storefront_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestToggleWishlistItem_addsAnAvailableProduct(t *testing.T) {
	wishlistRepo := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", SalesChannelID: "channel-1"},
	}}
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	uc := appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), wishlistRepo, channelRepo, overrideRepo, productRepo, &eventrecordtest.Outbox{})

	wishlist, err := uc.Execute(context.Background(), "token-1", "product-1")

	require.NoError(t, err)
	require.Equal(t, []string{"product-1"}, wishlist.ProductIDs)
	require.Equal(t, wishlistRepo.updated, wishlist)
}

func TestToggleWishlistItem_removesAnAlreadySavedProduct(t *testing.T) {
	wishlistRepo := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", SalesChannelID: "channel-1", ProductIDs: []string{"product-1"}},
	}}
	uc := appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), wishlistRepo, &fakeChannelRepo{}, &fakeOverrideRepo{}, &fakeProductRepo{}, &eventrecordtest.Outbox{})

	wishlist, err := uc.Execute(context.Background(), "token-1", "product-1")

	require.NoError(t, err)
	require.Empty(t, wishlist.ProductIDs)
}

func TestToggleWishlistItem_removingADeactivatedProductStillWorks(t *testing.T) {
	// The product no longer resolves via productRepo/channelRepo at all --
	// removal must still succeed since it never re-checks availability.
	wishlistRepo := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", SalesChannelID: "channel-1", ProductIDs: []string{"product-1", "product-2"}},
	}}
	uc := appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), wishlistRepo, &fakeChannelRepo{}, &fakeOverrideRepo{}, &fakeProductRepo{}, &eventrecordtest.Outbox{})

	wishlist, err := uc.Execute(context.Background(), "token-1", "product-1")

	require.NoError(t, err)
	require.Equal(t, []string{"product-2"}, wishlist.ProductIDs)
}

func TestToggleWishlistItem_rejectsUnavailableProductOnAdd(t *testing.T) {
	wishlistRepo := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", SalesChannelID: "channel-1"},
	}}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	uc := appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), wishlistRepo, channelRepo, &fakeOverrideRepo{}, &fakeProductRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "token-1", "missing-product")

	require.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}

func TestToggleWishlistItem_notFound(t *testing.T) {
	uc := appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), &fakeWishlistRepo{}, &fakeChannelRepo{}, &fakeOverrideRepo{}, &fakeProductRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing-token", "product-1")

	require.ErrorIs(t, err, storefrontDomain.ErrWishlistNotFound)
}
