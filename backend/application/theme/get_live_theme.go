package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetLiveThemeUseCase is the storefront's entry point: the channel's live
// version (settings + document index). Documents are then fetched by
// version id, which never changes content, so they cache forever.
type GetLiveThemeUseCase struct{ deps Deps }

func NewGetLiveThemeUseCase(deps Deps) *GetLiveThemeUseCase { return &GetLiveThemeUseCase{deps: deps} }

func (uc *GetLiveThemeUseCase) Execute(ctx context.Context, salesChannelID string) (*VersionView, error) {
	var out VersionView
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		if _, err := uc.deps.requireThemedChannel(ctx, salesChannelID); err != nil {
			return err
		}
		th, err := uc.deps.Themes.FindLiveByChannel(ctx, salesChannelID)
		if err != nil {
			return err
		}
		if th == nil || th.PublishedVersionID == nil {
			return theme.ErrNoLiveTheme
		}
		v, err := uc.deps.Versions.FindByID(ctx, *th.PublishedVersionID)
		if err != nil {
			return err
		}
		view, err := uc.deps.versionView(ctx, *th, *v)
		if err != nil {
			return err
		}
		out = *view
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getLiveTheme: %w", err)
	}
	return &out, nil
}
