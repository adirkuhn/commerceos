package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type PublishThemeInput struct {
	ThemeID string
	// ExpectedRevision is the draft revision the merchant looked at when
	// pressing Publish, so a concurrent edit can't slip in unreviewed.
	ExpectedRevision int
	Note             string
}

// PublishThemeUseCase seals the draft and makes it the channel's live
// storefront, in one transaction: if anything fails, the previously live
// version stays live. Every document is re-validated first, so a draft
// saved against an older component catalog can't go live broken.
type PublishThemeUseCase struct{ deps Deps }

func NewPublishThemeUseCase(deps Deps) *PublishThemeUseCase { return &PublishThemeUseCase{deps: deps} }

func (uc *PublishThemeUseCase) Execute(ctx context.Context, in PublishThemeInput) (*theme.Version, error) {
	var out theme.Version
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByIDForUpdate(ctx, in.ThemeID)
		if err != nil {
			return err
		}
		if _, err := uc.deps.requireThemedChannel(ctx, th.SalesChannelID); err != nil {
			return err
		}
		draft, err := uc.deps.Versions.FindDraft(ctx, th.ID, true)
		if err != nil {
			return err
		}
		if draft == nil {
			return theme.ErrNoDraft
		}
		if err := uc.validateDocuments(ctx, draft.ID); err != nil {
			return err
		}
		if err := theme.Publish(th, draft, in.ExpectedRevision, in.Note, uc.deps.Now()); err != nil {
			return err
		}
		if err := uc.deps.Versions.Update(ctx, draft); err != nil {
			return err
		}
		if err := uc.deps.Themes.ClearLive(ctx, th.SalesChannelID, th.ID); err != nil {
			return err
		}
		if err := uc.deps.Themes.Update(ctx, th); err != nil {
			return err
		}
		out = *draft
		return uc.deps.enqueue(ctx, events.ThemePublished{
			Base: uc.deps.eventBase(), TenantID: th.TenantID, ThemeID: th.ID, SalesChannelID: th.SalesChannelID,
			VersionID: draft.ID, VersionNumber: draft.Number,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("publishTheme: %w", err)
	}
	return &out, nil
}

func (uc *PublishThemeUseCase) validateDocuments(ctx context.Context, versionID string) error {
	index, err := uc.deps.Docs.List(ctx, versionID)
	if err != nil {
		return err
	}
	for _, entry := range index {
		doc, err := uc.deps.Docs.Find(ctx, versionID, entry.Kind, entry.Key)
		if err != nil {
			return err
		}
		if err := theme.ValidateTree(*doc, uc.deps.Catalog); err != nil {
			return fmt.Errorf("%s %q: %w", doc.Kind, doc.Key, err)
		}
	}
	return nil
}
