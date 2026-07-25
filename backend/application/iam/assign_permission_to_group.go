package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type AssignPermissionInput struct {
	GroupID string
	Rule    string
}

type AssignPermissionToGroupUseCase struct {
	db     *gorm.DB
	gRepo  iamDomain.GroupRepository
	gpRepo iamDomain.GroupPermissionRepository
}

func NewAssignPermissionToGroupUseCase(db *gorm.DB, gRepo iamDomain.GroupRepository, gpRepo iamDomain.GroupPermissionRepository) *AssignPermissionToGroupUseCase {
	return &AssignPermissionToGroupUseCase{db: db, gRepo: gRepo, gpRepo: gpRepo}
}

func (uc *AssignPermissionToGroupUseCase) Execute(ctx context.Context, in AssignPermissionInput) error {
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
				return nil // idempotent
			}
		}
		return uc.gpRepo.Assign(ctx, in.GroupID, rule)
	})
	if err != nil {
		return fmt.Errorf("assignPermissionToGroup: %w", err)
	}
	return nil
}
