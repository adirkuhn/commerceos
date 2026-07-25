package iam

import (
	"context"
	"fmt"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteGroupUseCase struct {
	db        *gorm.DB
	groupRepo iamDomain.GroupRepository
}

func NewDeleteGroupUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{db: db, groupRepo: groupRepo}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.groupRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.groupRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteGroup: %w", err)
	}
	return nil
}
