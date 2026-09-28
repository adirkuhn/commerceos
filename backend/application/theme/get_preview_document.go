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

func (uc *GetPreviewDocumentUseCase) Execute(ctx context.Context, themeID string, kind theme.DocumentKind, key string) (*theme.Document, error) {
	var out *theme.Document
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		_, v, err := uc.deps.previewVersion(ctx, themeID)
		if err != nil {
			return err
		}
		out, err = uc.deps.Docs.Find(ctx, v.ID, kind, key)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getPreviewDocument: %w", err)
	}
	return out, nil
}
