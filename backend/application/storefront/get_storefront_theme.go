package storefront

import (
	"context"

	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
)

// getThemePageUC is application/theme.GetThemePageUseCase, redeclared here
// (interface at point of use, per this codebase's ISP convention) so this
// package never imports application/theme's concrete type.
type getThemePageUC interface {
	Execute(ctx context.Context, salesChannelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error)
}

// GetStorefrontThemeUseCase is the public-storefront thin wrapper around
// the admin Theme Builder's GetThemePageUseCase -- see that type's doc
// comment. It exists to own the one thing the public API needs that the
// admin one doesn't: validating a slug that arrived as a raw path segment
// from an unauthenticated-until-the-key-check shopper request, before it
// ever reaches the shared use case.
type GetStorefrontThemeUseCase struct {
	getThemePage getThemePageUC
}

func NewGetStorefrontThemeUseCase(getThemePage getThemePageUC) *GetStorefrontThemeUseCase {
	return &GetStorefrontThemeUseCase{getThemePage: getThemePage}
}

func (uc *GetStorefrontThemeUseCase) Execute(ctx context.Context, salesChannelID string, slug string) (*themeDomain.ThemePage, error) {
	pageSlug := themeDomain.PageSlug(slug)
	if !pageSlug.Valid() {
		return nil, themeDomain.ErrInvalidPageSlug
	}
	return uc.getThemePage.Execute(ctx, salesChannelID, pageSlug)
}
