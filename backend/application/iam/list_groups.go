package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListGroupsUseCase struct {
	db        *gorm.DB
	groupRepo iamDomain.GroupRepository
}

func NewListGroupsUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository) *ListGroupsUseCase {
	return &ListGroupsUseCase{db: db, groupRepo: groupRepo}
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context) ([]iamDomain.Group, error) {
	var groups []iamDomain.Group
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		g, err := uc.groupRepo.List(ctx)
		if err != nil {
			return err
		}
		groups = g
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listGroups: %w", err)
	}
	return groups, nil
}
