package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RollbackThemeUseCase makes an older published version of the theme live
// again (and the theme itself the channel's live theme). The draft, if
// any, is left as it is.
type RollbackThemeUseCase struct{ deps Deps }

func NewRollbackThemeUseCase(deps Deps) *RollbackThemeUseCase {
	return &RollbackThemeUseCase{deps: deps}
}

func (uc *RollbackThemeUseCase) Execute(ctx context.Context, themeID, versionID string) (*theme.Theme, error) {
	var out theme.Theme
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByIDForUpdate(ctx, themeID)
		if err != nil {
			return err
		}
		if _, err := uc.deps.requireThemedChannel(ctx, th.SalesChannelID); err != nil {
			return err
		}
		target, err := uc.deps.Versions.FindByID(ctx, versionID)
		if err != nil {
			return err
		}
		if err := theme.Rollback(th, *target, uc.deps.Now()); err != nil {
			return err
		}
		if err := uc.deps.Themes.ClearLive(ctx, th.SalesChannelID, th.ID); err != nil {
			return err
		}
		if err := uc.deps.Themes.Update(ctx, th); err != nil {
			return err
		}
		out = *th
		return uc.deps.enqueue(ctx, events.ThemeRolledBack{
			Base: uc.deps.eventBase(), TenantID: th.TenantID, ThemeID: th.ID, SalesChannelID: th.SalesChannelID,
			VersionID: target.ID, VersionNumber: target.Number,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("rollbackTheme: %w", err)
	}
	return &out, nil
}
