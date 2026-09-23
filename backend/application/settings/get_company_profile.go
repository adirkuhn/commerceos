package settings

import (
	"context"
	"fmt"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetCompanyProfileUseCase struct {
	db   *gorm.DB
	repo settingsDomain.CompanyProfileRepository
}

func NewGetCompanyProfileUseCase(db *gorm.DB, repo settingsDomain.CompanyProfileRepository) *GetCompanyProfileUseCase {
	return &GetCompanyProfileUseCase{db: db, repo: repo}
}

// Execute returns the tenant's company profile -- an empty one (with the
// tenant set) if it hasn't been filled in yet.
func (uc *GetCompanyProfileUseCase) Execute(ctx context.Context) (*settingsDomain.CompanyProfile, error) {
	var result *settingsDomain.CompanyProfile
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.repo.Get(ctx)
		if err != nil {
			return err
		}
		p.TenantID = tenant.IDFromContext(ctx)
		result = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getCompanyProfile: %w", err)
	}
	return result, nil
}
