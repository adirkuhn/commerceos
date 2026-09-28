package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const (
	// MaxSitemapProducts stays under the sitemap protocol's 50,000 URLs per
	// file, leaving room for pages and collections.
	MaxSitemapProducts = 45000
	// maxSitemapCollectionPages x collectionsPerPage bounds collections.
	maxSitemapCollectionPages = 10
	collectionsPerPage        = 100
)

type CollectionLink struct {
	Slug      string
	UpdatedAt time.Time
}

// Sitemap is what a channel's storefront links to: its sellable products
// and the published collections.
type Sitemap struct {
	Products    []pmsDomain.ProductLink
	Collections []CollectionLink
}

type GetSitemapUseCase struct {
	db           *gorm.DB
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	products     pmsDomain.ProductLinkLister
	collections  collectionLister
}

func NewGetSitemapUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	products pmsDomain.ProductLinkLister,
	collections collectionLister,
) *GetSitemapUseCase {
	return &GetSitemapUseCase{db: db, channelRepo: channelRepo, overrideRepo: overrideRepo, products: products, collections: collections}
}

func (uc *GetSitemapUseCase) Execute(ctx context.Context, salesChannelID string) (*Sitemap, error) {
	var sm Sitemap
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := findInternalChannel(ctx, uc.channelRepo, salesChannelID)
		if err != nil {
			return err
		}
		overrides, err := uc.overrideRepo.ListByChannel(ctx, salesChannelID)
		if err != nil {
			return err
		}
		scope := channelEnrollment(channel, overrides)
		active := pmsDomain.ProductStatusActive
		sm.Products, err = uc.products.ListLinks(ctx, pmsDomain.ProductFilter{
			CategoryID: scope.categoryID, Tag: scope.tag, IncludeIDs: scope.includeIDs, ExcludeIDs: scope.excludeIDs,
			Status: &active, Limit: MaxSitemapProducts,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getSitemap: %w", err)
	}

	// Collections are listed by their own use case (its own transaction).
	for page := 1; page <= maxSitemapCollectionPages; page++ {
		list, total, err := uc.collections.Execute(ctx, true, page, collectionsPerPage)
		if err != nil {
			return nil, fmt.Errorf("getSitemap: %w", err)
		}
		for _, c := range list {
			sm.Collections = append(sm.Collections, CollectionLink{Slug: c.Slug, UpdatedAt: c.UpdatedAt})
		}
		if int64(page*collectionsPerPage) >= total || len(list) == 0 {
			break
		}
	}
	return &sm, nil
}
