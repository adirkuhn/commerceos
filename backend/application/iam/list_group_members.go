package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListGroupMembersUseCase struct {
	db            *gorm.DB
	groupRepo     iamDomain.GroupRepository
	userGroupRepo iamDomain.UserGroupRepository
}

func NewListGroupMembersUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository, userGroupRepo iamDomain.UserGroupRepository) *ListGroupMembersUseCase {
	return &ListGroupMembersUseCase{db: db, groupRepo: groupRepo, userGroupRepo: userGroupRepo}
}

func (uc *ListGroupMembersUseCase) Execute(ctx context.Context, groupID string) ([]string, error) {
	var members []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.groupRepo.FindByID(ctx, groupID); err != nil {
			return err
		}
		m, err := uc.userGroupRepo.ListMembers(ctx, groupID)
		if err != nil {
			return err
		}
		members = m
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listGroupMembers: %w", err)
	}
	return members, nil
}
