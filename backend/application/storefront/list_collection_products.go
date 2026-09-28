package storefront

import (
	"context"
	"fmt"
)

type collectionProductsResolver interface {
	Execute(ctx context.Context, collectionID string) ([]string, error)
}

type productLister interface {
	Execute(ctx context.Context, salesChannelID string, in ListProductsInput) (*ListProductsResult, error)
}

// ListCollectionProductsUseCase lists one collection's products on a
// channel: the collection decides which products and in what order
// (application/collections), the regular listing applies the channel's
// enrollment, pricing and pagination on top.
type ListCollectionProductsUseCase struct {
	resolve collectionProductsResolver
	list    productLister
}

func NewListCollectionProductsUseCase(resolve collectionProductsResolver, list productLister) *ListCollectionProductsUseCase {
	return &ListCollectionProductsUseCase{resolve: resolve, list: list}
}

func (uc *ListCollectionProductsUseCase) Execute(ctx context.Context, salesChannelID, collectionID string, in ListProductsInput) (*ListProductsResult, error) {
	ids, err := uc.resolve.Execute(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("listCollectionProducts: %w", err)
	}
	if ids == nil {
		ids = []string{} // non-nil: an empty collection matches nothing
	}
	in.ProductIDs, in.OrderByIDs = ids, ids
	return uc.list.Execute(ctx, salesChannelID, in)
}
