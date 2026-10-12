package settings

import (
	"context"
	"fmt"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetAppearanceUseCase struct {
	db   *gorm.DB
	repo settingsDomain.AppearanceRepository
}

func NewGetAppearanceUseCase(db *gorm.DB, repo settingsDomain.AppearanceRepository) *GetAppearanceUseCase {
	return &GetAppearanceUseCase{db: db, repo: repo}
}

// Execute returns the shop's admin appearance -- the defaults (zero
// UpdatedAt) if it was never saved.
func (uc *GetAppearanceUseCase) Execute(ctx context.Context) (*settingsDomain.Appearance, error) {
	var result settingsDomain.Appearance
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		a, err := currentAppearance(ctx, uc.repo)
		result = a
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getAppearance: %w", err)
	}
	return &result, nil
}

// currentAppearance is the saved appearance, or the defaults for the
// tenant in ctx. Runs inside tenant.WithTenant.
func currentAppearance(ctx context.Context, repo settingsDomain.AppearanceRepository) (settingsDomain.Appearance, error) {
	saved, err := repo.Get(ctx)
	if err != nil {
		return settingsDomain.Appearance{}, err
	}
	if saved == nil {
		return settingsDomain.DefaultAppearance(tenant.IDFromContext(ctx)), nil
	}
	return *saved, nil
}
