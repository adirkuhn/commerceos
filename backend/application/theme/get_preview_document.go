package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetPreviewDocumentUseCase struct{ deps Deps }

func NewGetPreviewDocumentUseCase(deps Deps) *GetPreviewDocumentUseCase {
	return &GetPreviewDocumentUseCase{deps: deps}
}

func (uc *GetPreviewDocumentUseCase) Execute(ctx context.Context, salesChannelID, themeID string, kind theme.DocumentKind, key string) (*DocumentView, error) {
	var out *DocumentView
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		_, v, err := uc.deps.previewVersion(ctx, salesChannelID, themeID)
		if err != nil {
			return err
		}
		doc, err := uc.deps.Docs.Find(ctx, v.ID, kind, key)
		if err != nil {
			return err
		}
		out, err = uc.deps.documentView(ctx, doc)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getPreviewDocument: %w", err)
	}
	return out, nil
}
