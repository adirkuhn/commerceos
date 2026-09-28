package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreateCollectionUseCase struct{ deps Deps }

func NewCreateCollectionUseCase(deps Deps) *CreateCollectionUseCase {
	return &CreateCollectionUseCase{deps: deps}
}

func (uc *CreateCollectionUseCase) Execute(ctx context.Context, d collections.Draft) (*collections.Collection, error) {
	var out collections.Collection
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		c, err := collections.New(tenant.IDFromContext(ctx), d, uc.deps.Now())
		if err != nil {
			return err
		}
		if err := uc.deps.Repo.Create(ctx, &c); err != nil {
			return err
		}
		out = c
		return uc.deps.changed(ctx, c, false)
	})
	if err != nil {
		return nil, fmt.Errorf("createCollection: %w", err)
	}
	return &out, nil
}
