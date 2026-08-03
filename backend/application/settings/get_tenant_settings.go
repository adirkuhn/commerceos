package settings

import (
	"context"
	"fmt"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetTenantSettingsUseCase struct {
	db   *gorm.DB
	repo settingsDomain.Repository
}

func NewGetTenantSettingsUseCase(db *gorm.DB, repo settingsDomain.Repository) *GetTenantSettingsUseCase {
	return &GetTenantSettingsUseCase{db: db, repo: repo}
}

func (uc *GetTenantSettingsUseCase) Execute(ctx context.Context) (*settingsDomain.TenantSettings, error) {
	var result settingsDomain.TenantSettings
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.repo.Get(ctx)
		if err != nil {
			return err
		}
		result = *s
		result.TenantID = tenant.IDFromContext(ctx)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getTenantSettings: %w", err)
	}
	return &result, nil
}
