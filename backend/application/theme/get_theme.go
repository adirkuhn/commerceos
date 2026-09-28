package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ThemeDetail is a theme with its current draft and published version
// (either may be nil).
type ThemeDetail struct {
	Theme     theme.Theme
	Draft     *theme.Version
	Published *theme.Version
}

type GetThemeUseCase struct{ deps Deps }

func NewGetThemeUseCase(deps Deps) *GetThemeUseCase { return &GetThemeUseCase{deps: deps} }

func (uc *GetThemeUseCase) Execute(ctx context.Context, id string) (*ThemeDetail, error) {
	var out ThemeDetail
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByID(ctx, id)
		if err != nil {
			return err
		}
		out.Theme = *th
		if out.Draft, err = uc.deps.Versions.FindDraft(ctx, id, false); err != nil {
			return err
		}
		if th.PublishedVersionID != nil {
			out.Published, err = uc.deps.Versions.FindByID(ctx, *th.PublishedVersionID)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getTheme: %w", err)
	}
	return &out, nil
}
