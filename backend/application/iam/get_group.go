package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetGroupUseCase struct {
	db        *gorm.DB
	groupRepo iamDomain.GroupRepository
}

func NewGetGroupUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository) *GetGroupUseCase {
	return &GetGroupUseCase{db: db, groupRepo: groupRepo}
}

func (uc *GetGroupUseCase) Execute(ctx context.Context, id string) (*iamDomain.Group, error) {
	var group *iamDomain.Group
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		g, err := uc.groupRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		group = g
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getGroup: %w", err)
	}
	return group, nil
}
