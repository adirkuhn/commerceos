package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListAssetsUseCase lists the tenant's theme media library, newest first.
type ListAssetsUseCase struct{ deps Deps }

func NewListAssetsUseCase(deps Deps) *ListAssetsUseCase { return &ListAssetsUseCase{deps: deps} }

func (uc *ListAssetsUseCase) Execute(ctx context.Context, page, perPage int) ([]theme.Asset, int64, error) {
	var list []theme.Asset
	var total int64
	limit, offset := pageWindow(page, perPage)
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		var err error
		list, total, err = uc.deps.Assets.ListActive(ctx, limit, offset)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listAssets: %w", err)
	}
	return list, total, nil
}
