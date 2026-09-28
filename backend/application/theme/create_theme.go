package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreateThemeInput struct {
	SalesChannelID string
	Name           string
	PresetKey      string
	UserID         string
}

// CreateThemeUseCase adds a theme to a channel's library with a first
// draft built from the chosen preset. It doesn't go live until published.
type CreateThemeUseCase struct{ deps Deps }

func NewCreateThemeUseCase(deps Deps) *CreateThemeUseCase { return &CreateThemeUseCase{deps: deps} }

func (uc *CreateThemeUseCase) Execute(ctx context.Context, in CreateThemeInput) (*theme.Theme, error) {
	var created theme.Theme
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		if _, err := uc.deps.requireThemedChannel(ctx, in.SalesChannelID); err != nil {
			return err
		}
		now := uc.deps.Now()
		th, err := theme.NewTheme(tenant.IDFromContext(ctx), in.SalesChannelID, in.Name, in.PresetKey, now)
		if err != nil {
			return err
		}
		docs, err := theme.PresetDocuments(in.PresetKey)
		if err != nil {
			return err
		}
		if err := uc.deps.Themes.Create(ctx, &th); err != nil {
			return err
		}
		draft := theme.NewDraftVersion(th, 1, theme.DefaultSettings(), in.UserID, now)
		if err := uc.deps.Versions.Create(ctx, &draft); err != nil {
			return err
		}
		for i := range docs {
			docs[i].TenantID, docs[i].VersionID = th.TenantID, draft.ID
			if err := uc.deps.Docs.Create(ctx, &docs[i]); err != nil {
				return err
			}
		}
		created = th
		return uc.deps.enqueue(ctx, events.ThemeCreated{
			Base: uc.deps.eventBase(), TenantID: th.TenantID, ThemeID: th.ID, SalesChannelID: th.SalesChannelID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createTheme: %w", err)
	}
	return &created, nil
}
