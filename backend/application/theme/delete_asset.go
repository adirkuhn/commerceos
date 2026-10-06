package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
		if err := uc.deps.Assets.Update(ctx, a); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemeAssetDeleted, a.ID, eventrecord.Data{"filename": a.Filename})
	})
	if err != nil {
		return fmt.Errorf("deleteAsset: %w", err)
	}
	return nil
}
