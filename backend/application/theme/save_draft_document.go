package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type SaveDraftDocumentInput struct {
	ThemeID          string
	ExpectedRevision int
	Kind             theme.DocumentKind
	Key              string
	// Title is kept for pages (and required there); templates and section
	// groups have no title.
	Title string
	SEO   theme.SEO
	Root  theme.Node
}

// DraftWrite is a changed document plus the draft's new revision, which the
// editor sends with its next write.
type DraftWrite struct {
	Document theme.Document
	Revision int
}

type SaveDraftDocumentUseCase struct{ deps Deps }

func NewSaveDraftDocumentUseCase(deps Deps) *SaveDraftDocumentUseCase {
	return &SaveDraftDocumentUseCase{deps: deps}
}

func (uc *SaveDraftDocumentUseCase) Execute(ctx context.Context, in SaveDraftDocumentInput) (*DraftWrite, error) {
	var out DraftWrite
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		draft, err := uc.deps.lockedDraft(ctx, in.ThemeID)
		if err != nil {
			return err
		}
		doc, err := uc.deps.Docs.Find(ctx, draft.ID, in.Kind, in.Key)
		if err != nil {
			return err
		}
		if doc.Kind == theme.KindPage {
			if doc.Title, err = theme.ValidPageTitle(in.Title); err != nil {
				return err
			}
		}
		doc.SEO, doc.Root = in.SEO, in.Root
		if err := theme.ValidateTree(*doc, uc.deps.Catalog); err != nil {
			return err
		}
		if err := draft.BeginEdit(in.ExpectedRevision); err != nil {
			return err
		}
		if err := uc.deps.Docs.Update(ctx, doc); err != nil {
			return err
		}
		out = DraftWrite{Document: *doc, Revision: draft.Revision}
		return uc.deps.Versions.Update(ctx, draft)
	})
	if err != nil {
		return nil, fmt.Errorf("saveDraftDocument: %w", err)
	}
	return &out, nil
}
