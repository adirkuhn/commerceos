package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListVersionsUseCase lists a theme's versions, newest first -- the
// history a merchant rolls back from.
type ListVersionsUseCase struct{ deps Deps }

func NewListVersionsUseCase(deps Deps) *ListVersionsUseCase { return &ListVersionsUseCase{deps: deps} }

func (uc *ListVersionsUseCase) Execute(ctx context.Context, themeID string, page, perPage int) ([]theme.Version, int64, error) {
	var list []theme.Version
	var total int64
	limit, offset := pageWindow(page, perPage)
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		if _, err := uc.deps.Themes.FindByID(ctx, themeID); err != nil {
			return err
		}
		var err error
		list, total, err = uc.deps.Versions.ListByTheme(ctx, themeID, limit, offset)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listVersions: %w", err)
	}
	return list, total, nil
}
