package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListGroupPermissionsUseCase struct {
	db     *gorm.DB
	gRepo  iamDomain.GroupRepository
	gpRepo iamDomain.GroupPermissionRepository
}

func NewListGroupPermissionsUseCase(db *gorm.DB, gRepo iamDomain.GroupRepository, gpRepo iamDomain.GroupPermissionRepository) *ListGroupPermissionsUseCase {
	return &ListGroupPermissionsUseCase{db: db, gRepo: gRepo, gpRepo: gpRepo}
}

func (uc *ListGroupPermissionsUseCase) Execute(ctx context.Context, groupID string) ([]iamDomain.Rule, error) {
	var rules []iamDomain.Rule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.gRepo.FindByID(ctx, groupID); err != nil {
			return err
		}
		r, err := uc.gpRepo.ListByGroup(ctx, groupID)
		if err != nil {
			return err
		}
		rules = r
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listGroupPermissions: %w", err)
	}
	return rules, nil
}
