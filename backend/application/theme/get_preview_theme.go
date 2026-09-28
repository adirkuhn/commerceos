package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetPreviewThemeUseCase is GetLiveThemeUseCase for a preview session:
// same shape, but the theme's draft. Read-only.
type GetPreviewThemeUseCase struct{ deps Deps }

func NewGetPreviewThemeUseCase(deps Deps) *GetPreviewThemeUseCase {
	return &GetPreviewThemeUseCase{deps: deps}
}

func (uc *GetPreviewThemeUseCase) Execute(ctx context.Context, themeID string) (*VersionView, error) {
	var out VersionView
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, v, err := uc.deps.previewVersion(ctx, themeID)
		if err != nil {
			return err
		}
		docs, err := uc.deps.Docs.List(ctx, v.ID)
		if err != nil {
			return err
		}
		out = VersionView{Theme: *th, Version: *v, Documents: docs}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPreviewTheme: %w", err)
	}
	return &out, nil
}
