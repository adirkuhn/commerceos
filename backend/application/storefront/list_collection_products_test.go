package storefront_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/collections"
)

func TestListProducts_orderByIDsPassesThrough(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{})

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{OrderByIDs: []string{"b", "a"}, Page: 1, PerPage: 20})

	require.NoError(t, err)
	assert.Equal(t, []string{"b", "a"}, catalog.gotIn.OrderByIDs)
}

type stubResolveCollection struct {
	ids []string
	err error
	got string
}

func (s *stubResolveCollection) Execute(_ context.Context, id string) ([]string, error) {
	s.got = id
	return s.ids, s.err
}

type stubListProducts struct {
	got     appstorefront.ListProductsInput
	channel string
}

func (s *stubListProducts) Execute(_ context.Context, channel string, in appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error) {
	s.got, s.channel = in, channel
	return &appstorefront.ListProductsResult{Total: 2}, nil
}

func TestListCollectionProducts_narrowsToTheCollectionInItsOrder(t *testing.T) {
	resolve := &stubResolveCollection{ids: []string{"p3", "p1"}}
	list := &stubListProducts{}
	uc := appstorefront.NewListCollectionProductsUseCase(resolve, list)

	res, err := uc.Execute(context.Background(), "channel-1", "col-1", appstorefront.ListProductsInput{Page: 2, PerPage: 12, Search: "red"})

	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)
	assert.Equal(t, "col-1", resolve.got)
	assert.Equal(t, "channel-1", list.channel)
	assert.Equal(t, []string{"p3", "p1"}, list.got.ProductIDs)
	assert.Equal(t, []string{"p3", "p1"}, list.got.OrderByIDs)
	assert.Equal(t, 2, list.got.Page)
	assert.Equal(t, "red", list.got.Search)
}

func TestListCollectionProducts_emptyCollectionMatchesNothing(t *testing.T) {
	list := &stubListProducts{}
	_, err := appstorefront.NewListCollectionProductsUseCase(&stubResolveCollection{ids: nil}, list).
		Execute(context.Background(), "channel-1", "col-1", appstorefront.ListProductsInput{})

	require.NoError(t, err)
	require.NotNil(t, list.got.ProductIDs, "non-nil empty list means 'match nothing'")
	assert.Empty(t, list.got.ProductIDs)
}

func TestListCollectionProducts_unknownCollection(t *testing.T) {
	_, err := appstorefront.NewListCollectionProductsUseCase(&stubResolveCollection{err: collections.ErrCollectionNotFound}, &stubListProducts{}).
		Execute(context.Background(), "channel-1", "col-1", appstorefront.ListProductsInput{})
	assert.True(t, errors.Is(err, collections.ErrCollectionNotFound))
}
