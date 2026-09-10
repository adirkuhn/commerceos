// Package theme holds the use cases behind the visual page-builder --
// fetching a channel's saved canvas document (read by both the admin
// editor and, via application/storefront's own thin wrapper, the public
// storefront API) and saving it (admin-only). Kept separate from
// application/channels/application/storefront for the same reason every
// other bounded concern in this codebase gets its own package: it's a
// distinct responsibility (page layout) from either sales-channel
// management or checkout.
package theme

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetThemePageUseCase struct {
	db   *gorm.DB
	repo themeDomain.Repository
}

func NewGetThemePageUseCase(db *gorm.DB, repo themeDomain.Repository) *GetThemePageUseCase {
	return &GetThemePageUseCase{db: db, repo: repo}
}

// Execute returns a default, unsaved ThemePage (empty Blocks, CanvasWidth
// 1440) when the channel has never saved this page -- a brand-new internal
// channel has no theme yet, and that's a normal state for the editor (an
// empty canvas to start from) and the storefront-frontend (falls back to
// each required functional block's own default rendering, see
// ThemePage.HasRequiredBlock) alike, not an error.
func (uc *GetThemePageUseCase) Execute(ctx context.Context, salesChannelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error) {
	var page *themeDomain.ThemePage
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.repo.FindBySlug(ctx, salesChannelID, slug)
		if err != nil {
			return err
		}
		if found == nil {
			page = &themeDomain.ThemePage{SalesChannelID: salesChannelID, Slug: slug, CanvasWidth: 1440, EditorMode: themeDomain.EditorModeFreeform}
			return nil
		}
		page = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getThemePage: %w", err)
	}
	return page, nil
}
