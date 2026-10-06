package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/locale"
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
	// Translations: the title/SEO in the store's other languages.
	Translations map[locale.Locale]theme.DocumentTexts
	Root         theme.Node
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
		translations, err := theme.ValidDocumentTranslations(doc.Kind, in.Translations)
		if err != nil {
			return err
		}
		doc.SEO, doc.Translations, doc.Root = in.SEO, translations, in.Root
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
		if err := uc.deps.Versions.Update(ctx, draft); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemeDocumentUpdated, doc.ID, eventrecord.Data{
			"theme_id": in.ThemeID, "version_id": draft.ID, "kind": string(doc.Kind), "key": doc.Key, "revision": draft.Revision,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("saveDraftDocument: %w", err)
	}
	return &out, nil
}
