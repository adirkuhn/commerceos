package settings

import (
	"context"
	"fmt"
	"time"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateTenantSettingsInput struct {
	DefaultCurrency   string
	Timezone          string
	DateFormat        string
	Language          string
	PricesIncludeTax  bool
	DefaultTaxCountry string
}

type UpdateTenantSettingsUseCase struct {
	db   *gorm.DB
	repo settingsDomain.Repository
}

func NewUpdateTenantSettingsUseCase(db *gorm.DB, repo settingsDomain.Repository) *UpdateTenantSettingsUseCase {
	return &UpdateTenantSettingsUseCase{db: db, repo: repo}
}

func (uc *UpdateTenantSettingsUseCase) Execute(ctx context.Context, in UpdateTenantSettingsInput) (*settingsDomain.TenantSettings, error) {
	var result settingsDomain.TenantSettings
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		result = settingsDomain.TenantSettings{
			TenantID:          tenant.IDFromContext(ctx),
			DefaultCurrency:   in.DefaultCurrency,
			Timezone:          in.Timezone,
			DateFormat:        in.DateFormat,
			Language:          in.Language,
			PricesIncludeTax:  in.PricesIncludeTax,
			DefaultTaxCountry: in.DefaultTaxCountry,
			UpdatedAt:         time.Now(),
		}
		return uc.repo.Upsert(ctx, &result)
	})
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}
	return &result, nil
}
