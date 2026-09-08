package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestCreateWishlist_startsAnEmptyWishlistOnTheChannel(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	wishlistRepo := &fakeWishlistRepo{}
	uc := appstorefront.NewCreateWishlistUseCase(newMockGormDB(t, 1), channelRepo, wishlistRepo)
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")

	wishlist, err := uc.Execute(ctx, "channel-1")

	require.NoError(t, err)
	require.Equal(t, "wishlist-1", wishlist.ID)
	require.Equal(t, "channel-1", wishlist.SalesChannelID)
	require.Equal(t, "tenant-1", wishlist.TenantID)
	require.NotEmpty(t, wishlist.Token)
	require.Empty(t, wishlist.ProductIDs)
}

func TestCreateWishlist_rejectsUnknownChannel(t *testing.T) {
	uc := appstorefront.NewCreateWishlistUseCase(newMockGormDB(t, 1), &fakeChannelRepo{}, &fakeWishlistRepo{})

	_, err := uc.Execute(context.Background(), "missing-channel")

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}
