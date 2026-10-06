package theme

import (
	"context"
	"errors"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreatePageInput struct {
	ThemeID          string
	ExpectedRevision int
	Key              string
	Title            string
}

type CreatePageUseCase struct{ deps Deps }

func NewCreatePageUseCase(deps Deps) *CreatePageUseCase { return &CreatePageUseCase{deps: deps} }

func (uc *CreatePageUseCase) Execute(ctx context.Context, in CreatePageInput) (*DraftWrite, error) {
	page, err := theme.NewPage(in.Key, in.Title)
	if err != nil {
		return nil, fmt.Errorf("createPage: %w", err)
	}
	var out DraftWrite
	err = tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		draft, err := uc.deps.lockedDraft(ctx, in.ThemeID)
		if err != nil {
			return err
		}
		if _, err := uc.deps.Docs.Find(ctx, draft.ID, theme.KindPage, in.Key); err == nil {
			return theme.ErrDocumentExists
		} else if !errors.Is(err, theme.ErrDocumentNotFound) {
			return err
		}
		if err := draft.BeginEdit(in.ExpectedRevision); err != nil {
			return err
		}
		page.TenantID, page.VersionID = draft.TenantID, draft.ID
		if err := uc.deps.Docs.Create(ctx, &page); err != nil {
			return err
		}
		out = DraftWrite{Document: page, Revision: draft.Revision}
		if err := uc.deps.Versions.Update(ctx, draft); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemePageCreated, page.ID, eventrecord.Data{
			"theme_id": in.ThemeID, "version_id": draft.ID, "key": page.Key, "title": page.Title, "revision": draft.Revision,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createPage: %w", err)
	}
	return &out, nil
}
