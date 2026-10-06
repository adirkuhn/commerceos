package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// UpdateAssetUseCase changes an image's description (alt text).
type UpdateAssetUseCase struct{ deps Deps }

func NewUpdateAssetUseCase(deps Deps) *UpdateAssetUseCase { return &UpdateAssetUseCase{deps: deps} }

func (uc *UpdateAssetUseCase) Execute(ctx context.Context, id, alt string) (*theme.Asset, error) {
	var out theme.Asset
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		a, err := uc.deps.Assets.FindByID(ctx, id)
		if err != nil {
			return err
		}
		before := a.Alt
		if err := a.SetAlt(alt); err != nil {
			return err
		}
		out = *a
		if err := uc.deps.Assets.Update(ctx, a); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemeAssetUpdated, a.ID,
			eventrecord.NewChanges().Set("alt", before, a.Alt))
	})
	if err != nil {
		return nil, fmt.Errorf("updateAsset: %w", err)
	}
	return &out, nil
}
