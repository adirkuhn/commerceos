package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UpdateCollectionUseCase struct{ deps Deps }

func NewUpdateCollectionUseCase(deps Deps) *UpdateCollectionUseCase {
	return &UpdateCollectionUseCase{deps: deps}
}

// Execute replaces the collection's settings. Turning a manual collection
// smart drops its hand-picked products (they'd never be read again).
func (uc *UpdateCollectionUseCase) Execute(ctx context.Context, id string, d collections.Draft) (*collections.Collection, error) {
	var out collections.Collection
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := uc.deps.Repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		wasManual := c.Type == collections.TypeManual
		if err := c.Update(d, uc.deps.Now()); err != nil {
			return err
		}
		if err := uc.deps.Repo.Update(ctx, c); err != nil {
			return err
		}
		if wasManual && c.Type != collections.TypeManual {
			if err := uc.deps.Repo.SetManualProducts(ctx, c.ID, nil); err != nil {
				return err
			}
		}
		out = *c
		return uc.deps.changed(ctx, *c, false)
	})
	if err != nil {
		return nil, fmt.Errorf("updateCollection: %w", err)
	}
	return &out, nil
}
