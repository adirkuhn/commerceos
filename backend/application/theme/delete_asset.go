package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DeleteAssetUseCase removes an image from the library. The file and row
// stay (soft delete): published theme versions may still show it.
type DeleteAssetUseCase struct{ deps Deps }

func NewDeleteAssetUseCase(deps Deps) *DeleteAssetUseCase { return &DeleteAssetUseCase{deps: deps} }

func (uc *DeleteAssetUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		a, err := uc.deps.Assets.FindByID(ctx, id)
		if err != nil {
			return err
		}
		now := uc.deps.Now()
		a.DeletedAt = &now
		return uc.deps.Assets.Update(ctx, a)
	})
	if err != nil {
		return fmt.Errorf("deleteAsset: %w", err)
	}
	return nil
}
