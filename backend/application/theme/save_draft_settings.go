package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type SaveDraftSettingsInput struct {
	ThemeID          string
	ExpectedRevision int
	Settings         theme.Settings
}

type SaveDraftSettingsUseCase struct{ deps Deps }

func NewSaveDraftSettingsUseCase(deps Deps) *SaveDraftSettingsUseCase {
	return &SaveDraftSettingsUseCase{deps: deps}
}

func (uc *SaveDraftSettingsUseCase) Execute(ctx context.Context, in SaveDraftSettingsInput) (*theme.Version, error) {
	if err := in.Settings.Validate(); err != nil {
		return nil, fmt.Errorf("saveDraftSettings: %w", err)
	}
	var out theme.Version
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		draft, err := uc.deps.lockedDraft(ctx, in.ThemeID)
		if err != nil {
			return err
		}
		if err := draft.BeginEdit(in.ExpectedRevision); err != nil {
			return err
		}
		draft.Settings = in.Settings
		out = *draft
		if err := uc.deps.Versions.Update(ctx, draft); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemeSettingsUpdated, in.ThemeID, eventrecord.Data{
			"version_id": draft.ID, "revision": draft.Revision,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("saveDraftSettings: %w", err)
	}
	return &out, nil
}
