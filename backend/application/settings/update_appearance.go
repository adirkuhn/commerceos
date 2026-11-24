package settings

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateAppearanceUseCase struct {
	db     *gorm.DB
	repo   settingsDomain.AppearanceRepository
	outbox eventrecord.Enqueuer
}

func NewUpdateAppearanceUseCase(db *gorm.DB, repo settingsDomain.AppearanceRepository, outbox eventrecord.Enqueuer) *UpdateAppearanceUseCase {
	return &UpdateAppearanceUseCase{db: db, repo: repo, outbox: outbox}
}

// Execute replaces the shop's admin appearance and records
// settings.appearance.updated in the same transaction.
func (uc *UpdateAppearanceUseCase) Execute(ctx context.Context, in settingsDomain.AppearanceInput) (*settingsDomain.Appearance, error) {
	var result *settingsDomain.Appearance
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		tenantID := tenant.IDFromContext(ctx)
		after, err := settingsDomain.NewAppearance(tenantID, in, time.Now().UTC())
		if err != nil {
			return err
		}
		before, err := currentAppearance(ctx, uc.repo)
		if err != nil {
			return err
		}
		if err := uc.repo.Upsert(ctx, after); err != nil {
			return err
		}
		result = after
		return eventrecord.Record(ctx, uc.outbox, events.AppearanceUpdated, tenantID, appearanceChanges(before, *after))
	})
	if err != nil {
		return nil, fmt.Errorf("updateAppearance: %w", err)
	}
	return result, nil
}

func appearanceChanges(before, after settingsDomain.Appearance) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("default_look", string(before.DefaultLook), string(after.DefaultLook)).
		Set("corners", string(before.Corners), string(after.Corners)).
		Set("text_size", string(before.TextSize), string(after.TextSize)).
		Set("density", string(before.Density), string(after.Density))
}
