package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SetCollectionProductsUseCase replaces a manual collection's products,
// in the given order.
type SetCollectionProductsUseCase struct{ deps Deps }

func NewSetCollectionProductsUseCase(deps Deps) *SetCollectionProductsUseCase {
	return &SetCollectionProductsUseCase{deps: deps}
}

func (uc *SetCollectionProductsUseCase) Execute(ctx context.Context, id string, productIDs []string) error {
	if err := collections.ValidateManualProducts(productIDs); err != nil {
		return fmt.Errorf("setCollectionProducts: %w", err)
	}
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := uc.deps.Repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if c.Type != collections.TypeManual {
			return collections.ErrNotManual
		}
		if err := uc.deps.Repo.SetManualProducts(ctx, id, productIDs); err != nil {
			return err
		}
		return uc.deps.changed(ctx, *c, false)
	})
	if err != nil {
		return fmt.Errorf("setCollectionProducts: %w", err)
	}
	return nil
}
