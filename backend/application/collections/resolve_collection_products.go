package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveCollectionProductsUseCase lists a published collection's active
// products in its sort order (at most MaxResolvedProducts). The storefront
// narrows these to the products enrolled on its channel.
type ResolveCollectionProductsUseCase struct{ deps Deps }

func NewResolveCollectionProductsUseCase(deps Deps) *ResolveCollectionProductsUseCase {
	return &ResolveCollectionProductsUseCase{deps: deps}
}

func (uc *ResolveCollectionProductsUseCase) Execute(ctx context.Context, collectionID string) ([]string, error) {
	var ids []string
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := uc.deps.Repo.FindByID(ctx, collectionID)
		if err != nil {
			return err
		}
		if !c.Published {
			return collections.ErrCollectionNotFound
		}
		ids, err = uc.deps.Repo.ResolveProductIDs(ctx, *c, MaxResolvedProducts)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("resolveCollectionProducts: %w", err)
	}
	return ids, nil
}
