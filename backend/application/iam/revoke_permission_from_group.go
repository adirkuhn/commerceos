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

type RevokePermissionInput struct {
	GroupID string
	Rule    string
}

type RevokePermissionFromGroupUseCase struct {
	db     *gorm.DB
	gRepo  iamDomain.GroupRepository
	gpRepo iamDomain.GroupPermissionRepository
	outbox eventrecord.Enqueuer
}

func NewRevokePermissionFromGroupUseCase(db *gorm.DB, gRepo iamDomain.GroupRepository, gpRepo iamDomain.GroupPermissionRepository, outbox eventrecord.Enqueuer) *RevokePermissionFromGroupUseCase {
	return &RevokePermissionFromGroupUseCase{db: db, gRepo: gRepo, gpRepo: gpRepo, outbox: outbox}
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
				if err := uc.gpRepo.Revoke(ctx, in.GroupID, rule); err != nil {
					return err
				}
				return eventrecord.Record(ctx, uc.outbox, events.GroupPermissionRevoked, in.GroupID, eventrecord.Data{"rule": string(rule)})
			}
		}
		return iamDomain.ErrPermissionNotAssigned
	})
	if err != nil {
		return fmt.Errorf("revokePermissionFromGroup: %w", err)
	}
	return nil
}
