package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetPublishedDocumentUseCase serves one document of a published version
// to the storefront. Any published version of one of the channel's themes
// qualifies (not only the live one), so a page rendered just before a
// publish can still load its documents; drafts never do.
type GetPublishedDocumentUseCase struct{ deps Deps }

func NewGetPublishedDocumentUseCase(deps Deps) *GetPublishedDocumentUseCase {
	return &GetPublishedDocumentUseCase{deps: deps}
}

func (uc *GetPublishedDocumentUseCase) Execute(ctx context.Context, salesChannelID, versionID string, kind theme.DocumentKind, key string) (*DocumentView, error) {
	var out *DocumentView
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		if _, err := uc.deps.requireThemedChannel(ctx, salesChannelID); err != nil {
			return err
		}
		v, err := uc.deps.Versions.FindByID(ctx, versionID)
		if err != nil {
			return err
		}
		if v.IsDraft() {
			return theme.ErrVersionNotFound
		}
		th, err := uc.deps.Themes.FindByID(ctx, v.ThemeID)
		if err != nil {
			return err
		}
		if th.SalesChannelID != salesChannelID {
			return theme.ErrVersionNotFound
		}
		doc, err := uc.deps.Docs.Find(ctx, v.ID, kind, key)
		if err != nil {
			return err
		}
		out, err = uc.deps.documentView(ctx, doc)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getPublishedDocument: %w", err)
	}
	return out, nil
}
