package iam

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

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
	outbox eventrecord.Enqueuer
}

func NewAssignPermissionToGroupUseCase(db *gorm.DB, gRepo iamDomain.GroupRepository, gpRepo iamDomain.GroupPermissionRepository, outbox eventrecord.Enqueuer) *AssignPermissionToGroupUseCase {
	return &AssignPermissionToGroupUseCase{db: db, gRepo: gRepo, gpRepo: gpRepo, outbox: outbox}
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
		if err := uc.gpRepo.Assign(ctx, in.GroupID, rule); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.GroupPermissionAssigned, in.GroupID, eventrecord.Data{"rule": string(rule)})
	})
	if err != nil {
		return fmt.Errorf("assignPermissionToGroup: %w", err)
	}
	return nil
}
