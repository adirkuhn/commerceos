package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type RevokePermissionInput struct {
	GroupID string
	Rule    string
}

type RevokePermissionFromGroupUseCase struct {
	db     *gorm.DB
	gRepo  iamDomain.GroupRepository
	gpRepo iamDomain.GroupPermissionRepository
}

func NewRevokePermissionFromGroupUseCase(db *gorm.DB, gRepo iamDomain.GroupRepository, gpRepo iamDomain.GroupPermissionRepository) *RevokePermissionFromGroupUseCase {
	return &RevokePermissionFromGroupUseCase{db: db, gRepo: gRepo, gpRepo: gpRepo}
}

func (uc *RevokePermissionFromGroupUseCase) Execute(ctx context.Context, in RevokePermissionInput) error {
	rule := iamDomain.Rule(in.Rule)
	if err := rule.Validate(); err != nil {
		return err
	}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.gRepo.FindByID(ctx, in.GroupID); err != nil {
			return err
		}
		rules, err := uc.gpRepo.ListByGroup(ctx, in.GroupID)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if r == rule {
				return uc.gpRepo.Revoke(ctx, in.GroupID, rule)
			}
		}
		return iamDomain.ErrPermissionNotAssigned
	})
	if err != nil {
		return fmt.Errorf("revokePermissionFromGroup: %w", err)
	}
	return nil
}
