package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// VersionView is a version plus its document index (documents without
// their trees -- the editor loads one document at a time).
type VersionView struct {
	Theme     theme.Theme
	Version   theme.Version
	Documents []theme.Document
	// Assets the settings refer to (logo, favicon), keyed by id.
	Assets map[string]theme.Asset
}

func (d Deps) versionView(ctx context.Context, th theme.Theme, v theme.Version) (*VersionView, error) {
	docs, err := d.Docs.List(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	assets, err := d.resolveAssets(ctx, v.Settings.ReferencedAssetIDs())
	if err != nil {
		return nil, err
	}
	return &VersionView{Theme: th, Version: v, Documents: docs, Assets: assets}, nil
}

// OpenDraftUseCase is what the editor calls on load: it returns the
// theme's draft, starting one from the published version when the last
// draft was published (a copy of its settings and documents).
type OpenDraftUseCase struct{ deps Deps }

func NewOpenDraftUseCase(deps Deps) *OpenDraftUseCase { return &OpenDraftUseCase{deps: deps} }

func (uc *OpenDraftUseCase) Execute(ctx context.Context, themeID, userID string) (*VersionView, error) {
	var out VersionView
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByIDForUpdate(ctx, themeID)
		if err != nil {
			return err
		}
		draft, err := uc.deps.Versions.FindDraft(ctx, themeID, true)
		if err != nil {
			return err
		}
		if draft == nil {
			if draft, err = uc.startDraft(ctx, *th, userID); err != nil {
				return err
			}
		}
		view, err := uc.deps.versionView(ctx, *th, *draft)
		if err != nil {
			return err
		}
		out = *view
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("openDraft: %w", err)
	}
	return &out, nil
}

func (uc *OpenDraftUseCase) startDraft(ctx context.Context, th theme.Theme, userID string) (*theme.Version, error) {
	if th.PublishedVersionID == nil {
		// Every theme is created with a draft, so no draft and nothing
		// published means the data is inconsistent.
		return nil, theme.ErrNoDraft
	}
	published, err := uc.deps.Versions.FindByID(ctx, *th.PublishedVersionID)
	if err != nil {
		return nil, err
	}
	number, err := uc.deps.Versions.NextNumber(ctx, th.ID)
	if err != nil {
		return nil, err
	}
	draft := theme.NextDraftFrom(*published, number, userID, uc.deps.Now())
	if err := uc.deps.Versions.Create(ctx, &draft); err != nil {
		return nil, err
	}
	if err := uc.deps.Docs.CopyAll(ctx, published.ID, draft.ID); err != nil {
		return nil, err
	}
	return &draft, nil
}
