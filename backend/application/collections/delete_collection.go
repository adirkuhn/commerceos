package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteCollectionUseCase struct{ deps Deps }

func NewDeleteCollectionUseCase(deps Deps) *DeleteCollectionUseCase {
	return &DeleteCollectionUseCase{deps: deps}
}

func (uc *DeleteCollectionUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := uc.deps.Repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.deps.Repo.Delete(ctx, id); err != nil {
			return err
		}
		return uc.deps.changed(ctx, *c, true)
	})
	if err != nil {
		return fmt.Errorf("deleteCollection: %w", err)
	}
	return nil
}
