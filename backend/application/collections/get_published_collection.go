package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetPublishedCollectionUseCase is the storefront read: an unpublished
// collection is indistinguishable from a missing one.
type GetPublishedCollectionUseCase struct{ deps Deps }

func NewGetPublishedCollectionUseCase(deps Deps) *GetPublishedCollectionUseCase {
	return &GetPublishedCollectionUseCase{deps: deps}
}

func (uc *GetPublishedCollectionUseCase) Execute(ctx context.Context, slug string) (*collections.Collection, error) {
	var out *collections.Collection
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := uc.deps.Repo.FindBySlug(ctx, slug)
		if err != nil {
			return err
		}
		if !c.Published {
			return collections.ErrCollectionNotFound
		}
		out = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPublishedCollection: %w", err)
	}
	return out, nil
}
