package storefront_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/collections"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

type stubLinkLister struct {
	got   pmsDomain.ProductFilter
	links []pmsDomain.ProductLink
}

func (s *stubLinkLister) ListLinks(_ context.Context, f pmsDomain.ProductFilter) ([]pmsDomain.ProductLink, error) {
	s.got = f
	return s.links, nil
}

// pagedCollections serves `total` published collections 100 per page.
type pagedCollections struct {
	total int
	pages []int
}

func (p *pagedCollections) Execute(_ context.Context, published bool, page, perPage int) ([]collections.Collection, int64, error) {
	p.pages = append(p.pages, page)
	var out []collections.Collection
	for i := (page - 1) * perPage; i < page*perPage && i < p.total; i++ {
		c := collections.Collection{ID: "c"}
		c.Slug = "col"
		c.Published = published
		out = append(out, c)
	}
	return out, int64(p.total), nil
}

func TestSitemap_listsTheChannelsActiveProductsAndPublishedCollections(t *testing.T) {
	categoryID := "cat-1"
	channel := internalChannel("channel-1")
	channel.EnrollmentMode = channelsDomain.EnrollmentModeCategory
	channel.EnrollmentCategoryID = &categoryID
	overrides := &fakeOverrideRepo{byChannel: map[string][]channelsDomain.SalesChannelProductOverride{"channel-1": {
		{ProductID: "p-in", Included: true}, {ProductID: "p-out", Included: false},
	}}}
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	links := &stubLinkLister{links: []pmsDomain.ProductLink{{Slug: "red-shoes", UpdatedAt: at}}}
	cols := &pagedCollections{total: 150}
	uc := appstorefront.NewGetSitemapUseCase(newMockGormDB(t, 1), &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": channel}}, overrides, links, cols)

	sm, err := uc.Execute(context.Background(), "channel-1")

	require.NoError(t, err)
	assert.Equal(t, []pmsDomain.ProductLink{{Slug: "red-shoes", UpdatedAt: at}}, sm.Products)
	assert.Len(t, sm.Collections, 150)
	assert.Equal(t, []int{1, 2}, cols.pages)
	require.NotNil(t, links.got.Status)
	assert.Equal(t, pmsDomain.ProductStatusActive, *links.got.Status)
	assert.Equal(t, &categoryID, links.got.CategoryID)
	assert.Equal(t, []string{"p-in"}, *links.got.IncludeIDs)
	assert.Equal(t, []string{"p-out"}, links.got.ExcludeIDs)
	assert.Equal(t, appstorefront.MaxSitemapProducts, links.got.Limit)
}

func TestSitemap_unknownChannel(t *testing.T) {
	uc := appstorefront.NewGetSitemapUseCase(newMockGormDB(t, 1), &fakeChannelRepo{}, &fakeOverrideRepo{}, &stubLinkLister{}, &pagedCollections{})
	_, err := uc.Execute(context.Background(), "nope")
	assert.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}
