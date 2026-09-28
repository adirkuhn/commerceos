package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListThemesUseCase lists a channel's theme library, live theme first.
type ListThemesUseCase struct{ deps Deps }

func NewListThemesUseCase(deps Deps) *ListThemesUseCase { return &ListThemesUseCase{deps: deps} }

func (uc *ListThemesUseCase) Execute(ctx context.Context, salesChannelID string, page, perPage int) ([]theme.Theme, int64, error) {
	var list []theme.Theme
	var total int64
	limit, offset := pageWindow(page, perPage)
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		var err error
		list, total, err = uc.deps.Themes.ListByChannel(ctx, salesChannelID, limit, offset)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listThemes: %w", err)
	}
	return list, total, nil
}
