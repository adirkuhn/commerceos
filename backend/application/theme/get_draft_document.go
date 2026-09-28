package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetDraftDocumentUseCase struct{ deps Deps }

func NewGetDraftDocumentUseCase(deps Deps) *GetDraftDocumentUseCase {
	return &GetDraftDocumentUseCase{deps: deps}
}

func (uc *GetDraftDocumentUseCase) Execute(ctx context.Context, themeID string, kind theme.DocumentKind, key string) (*DocumentView, error) {
	var out *DocumentView
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		draft, err := uc.deps.Versions.FindDraft(ctx, themeID, false)
		if err != nil {
			return err
		}
		if draft == nil {
			return theme.ErrNoDraft
		}
		doc, err := uc.deps.Docs.Find(ctx, draft.ID, kind, key)
		if err != nil {
			return err
		}
		out, err = uc.deps.documentView(ctx, doc)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getDraftDocument: %w", err)
	}
	return out, nil
}
