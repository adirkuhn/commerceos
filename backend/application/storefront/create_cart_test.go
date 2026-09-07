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

func TestCreateCart_startsAnEmptyCartOnTheChannel(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{}
	uc := appstorefront.NewCreateCartUseCase(newMockGormDB(t, 1), channelRepo, cartRepo)
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")

	cart, err := uc.Execute(ctx, "channel-1")

	require.NoError(t, err)
	require.Equal(t, "cart-1", cart.ID)
	require.Equal(t, "channel-1", cart.SalesChannelID)
	require.Equal(t, "tenant-1", cart.TenantID)
	require.NotEmpty(t, cart.Token)
	require.Empty(t, cart.Items)
	require.Equal(t, cartRepo.created, cart)
}

func TestCreateCart_rejectsUnknownChannel(t *testing.T) {
	uc := appstorefront.NewCreateCartUseCase(newMockGormDB(t, 1), &fakeChannelRepo{}, &fakeCartRepo{})

	_, err := uc.Execute(context.Background(), "missing-channel")

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}

func TestCreateCart_rejectsNonInternalChannel(t *testing.T) {
	wooChannel := channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, Active: true}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": wooChannel}}
	uc := appstorefront.NewCreateCartUseCase(newMockGormDB(t, 1), channelRepo, &fakeCartRepo{})

	_, err := uc.Execute(context.Background(), "channel-1")

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}

func TestCreateCart_rejectsInactiveChannel(t *testing.T) {
	inactive := internalChannel("channel-1")
	inactive.Active = false
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": inactive}}
	uc := appstorefront.NewCreateCartUseCase(newMockGormDB(t, 1), channelRepo, &fakeCartRepo{})

	_, err := uc.Execute(context.Background(), "channel-1")

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}
