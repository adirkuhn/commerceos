package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DeletePageUseCase removes a page from the draft. Templates and section
// groups can't be deleted -- every theme needs them. Returns the draft's
// new revision.
type DeletePageUseCase struct{ deps Deps }

func NewDeletePageUseCase(deps Deps) *DeletePageUseCase { return &DeletePageUseCase{deps: deps} }

func (uc *DeletePageUseCase) Execute(ctx context.Context, themeID string, expectedRevision int, key string) (int, error) {
	var revision int
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		draft, err := uc.deps.lockedDraft(ctx, themeID)
		if err != nil {
			return err
		}
		doc, err := uc.deps.Docs.Find(ctx, draft.ID, theme.KindPage, key)
		if err != nil {
			return err
		}
		if err := draft.BeginEdit(expectedRevision); err != nil {
			return err
		}
		if err := uc.deps.Docs.Delete(ctx, doc.ID); err != nil {
			return err
		}
		revision = draft.Revision
		if err := uc.deps.Versions.Update(ctx, draft); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemePageDeleted, doc.ID, eventrecord.Data{
			"theme_id": themeID, "version_id": draft.ID, "key": doc.Key, "revision": draft.Revision,
		})
	})
	if err != nil {
		return 0, fmt.Errorf("deletePage: %w", err)
	}
	return revision, nil
}
