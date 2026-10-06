package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type RenameThemeUseCase struct{ deps Deps }

func NewRenameThemeUseCase(deps Deps) *RenameThemeUseCase { return &RenameThemeUseCase{deps: deps} }

func (uc *RenameThemeUseCase) Execute(ctx context.Context, id, name string) (*theme.Theme, error) {
	var out theme.Theme
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByID(ctx, id)
		if err != nil {
			return err
		}
		before := th.Name
		if err := th.Rename(name, uc.deps.Now()); err != nil {
			return err
		}
		out = *th
		if err := uc.deps.Themes.Update(ctx, th); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.ThemeRenamed, th.ID,
			eventrecord.NewChanges().Set("name", before, th.Name))
	})
	if err != nil {
		return nil, fmt.Errorf("renameTheme: %w", err)
	}
	return &out, nil
}
