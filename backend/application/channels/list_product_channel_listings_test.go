package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestListProductChannelListings_returnsEveryRowForThisProductAndConnection(t *testing.T) {
	variantA, variantB := "variant-a", "variant-b"
	repo := &stubListingRepo{listByProduct: map[string][]channelsDomain.ProductChannelListing{
		"prod-1": {
			{ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", ExternalID: "1", VariantID: &variantA},
			{ID: "listing-2", ProductID: "prod-1", ConnectionID: "conn-1", ExternalID: "2", VariantID: &variantB},
			{ID: "listing-3", ProductID: "prod-1", ConnectionID: "conn-2", ExternalID: "3"}, // different connection -- excluded
		},
	}}
	uc := appchannels.NewListProductChannelListingsUseCase(newMockGormDB(t, 1), repo)

	listings, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	require.Len(t, listings, 2)
	ids := []string{listings[0].ID, listings[1].ID}
	require.ElementsMatch(t, []string{"listing-1", "listing-2"}, ids)
}

func TestListProductChannelListings_returnsEmptyWhenNotLinked(t *testing.T) {
	repo := &stubListingRepo{}
	uc := appchannels.NewListProductChannelListingsUseCase(newMockGormDB(t, 1), repo)

	listings, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	require.Empty(t, listings)
}

func TestListProductChannelListings_propagatesRepoError(t *testing.T) {
	repo := &stubListingRepo{listByProdErr: errors.New("db down")}
	uc := appchannels.NewListProductChannelListingsUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.Error(t, err)
}
