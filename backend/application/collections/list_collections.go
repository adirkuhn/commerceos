package collections

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListCollectionsUseCase lists collections by name; publishedOnly for shoppers.
type ListCollectionsUseCase struct{ deps Deps }

func NewListCollectionsUseCase(deps Deps) *ListCollectionsUseCase {
	return &ListCollectionsUseCase{deps: deps}
}

func (uc *ListCollectionsUseCase) Execute(ctx context.Context, publishedOnly bool, page, perPage int) ([]collections.Collection, int64, error) {
	var list []collections.Collection
	var total int64
	limit, offset := pageWindow(page, perPage)
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		var err error
		list, total, err = uc.deps.Repo.List(ctx, publishedOnly, limit, offset)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listCollections: %w", err)
	}
	return list, total, nil
}
