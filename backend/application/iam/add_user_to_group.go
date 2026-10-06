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

type AddUserToGroupInput struct {
	GroupID string
	UserID  string
}

type AddUserToGroupUseCase struct {
	db            *gorm.DB
	groupRepo     iamDomain.GroupRepository
	userGroupRepo iamDomain.UserGroupRepository
	outbox        eventrecord.Enqueuer
}

func NewAddUserToGroupUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository, userGroupRepo iamDomain.UserGroupRepository, outbox eventrecord.Enqueuer) *AddUserToGroupUseCase {
	return &AddUserToGroupUseCase{db: db, groupRepo: groupRepo, userGroupRepo: userGroupRepo, outbox: outbox}
}

func (uc *AddUserToGroupUseCase) Execute(ctx context.Context, in AddUserToGroupInput) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.groupRepo.FindByID(ctx, in.GroupID); err != nil {
			return err
		}
		isMember, err := uc.userGroupRepo.IsMember(ctx, in.GroupID, in.UserID)
		if err != nil {
			return err
		}
		if isMember {
			return iamDomain.ErrAlreadyMember
		}
		if err := uc.userGroupRepo.AddMember(ctx, in.GroupID, in.UserID); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.GroupMemberAdded, in.GroupID, eventrecord.Data{"user_id": in.UserID})
	})
	if err != nil {
		return fmt.Errorf("addUserToGroup: %w", err)
	}
	return nil
}
