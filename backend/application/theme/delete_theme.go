package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DeleteThemeUseCase removes a theme and all its versions. The live theme
// can't be deleted -- the storefront would have nothing to show.
type DeleteThemeUseCase struct{ deps Deps }

func NewDeleteThemeUseCase(deps Deps) *DeleteThemeUseCase { return &DeleteThemeUseCase{deps: deps} }

func (uc *DeleteThemeUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if err := th.CanDelete(); err != nil {
			return err
		}
		if err := uc.deps.Themes.Delete(ctx, id); err != nil {
			return err
		}
		return uc.deps.enqueue(ctx, events.ThemeDeleted{
			Base: uc.deps.eventBase(), TenantID: th.TenantID, ThemeID: th.ID, SalesChannelID: th.SalesChannelID,
		})
	})
	if err != nil {
		return fmt.Errorf("deleteTheme: %w", err)
	}
	return nil
}
