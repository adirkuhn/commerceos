package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// CollectionDetail is a collection plus, for manual ones, its products in order.
type CollectionDetail struct {
	collections.Collection
	ProductIDs []string
}

type GetCollectionUseCase struct{ deps Deps }

func NewGetCollectionUseCase(deps Deps) *GetCollectionUseCase {
	return &GetCollectionUseCase{deps: deps}
}

func (uc *GetCollectionUseCase) Execute(ctx context.Context, id string) (*CollectionDetail, error) {
	var out CollectionDetail
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := uc.deps.Repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		out.Collection = *c
		if c.Type == collections.TypeManual {
			out.ProductIDs, err = uc.deps.Repo.ManualProducts(ctx, id)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getCollection: %w", err)
	}
	return &out, nil
}
