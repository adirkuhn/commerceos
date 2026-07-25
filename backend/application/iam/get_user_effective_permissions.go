package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetUserEffectivePermissionsUseCase struct {
	db     *gorm.DB
	gpRepo iamDomain.GroupPermissionRepository
}

func NewGetUserEffectivePermissionsUseCase(db *gorm.DB, gpRepo iamDomain.GroupPermissionRepository) *GetUserEffectivePermissionsUseCase {
	return &GetUserEffectivePermissionsUseCase{db: db, gpRepo: gpRepo}
}

func (uc *GetUserEffectivePermissionsUseCase) Execute(ctx context.Context, userID string) ([]iamDomain.Rule, error) {
	var rules []iamDomain.Rule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.gpRepo.ListByUser(ctx, userID)
		if err != nil {
			return err
		}
		rules = r
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getUserEffectivePermissions: %w", err)
	}
	return rules, nil
}
