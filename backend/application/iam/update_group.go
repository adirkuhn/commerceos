package iam

import (
	"context"
	"errors"
	"fmt"
	"time"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateGroupInput struct {
	ID          string
	Name        string
	Description string
}

type UpdateGroupUseCase struct {
	db        *gorm.DB
	groupRepo iamDomain.GroupRepository
}

func NewUpdateGroupUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{db: db, groupRepo: groupRepo}
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, in UpdateGroupInput) (*iamDomain.Group, error) {
	var group *iamDomain.Group
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		g, err := uc.groupRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}

		// Check name uniqueness only when name changed
		if in.Name != g.Name {
			_, err := uc.groupRepo.FindByName(ctx, in.Name)
			if err == nil {
				return iamDomain.ErrGroupNameTaken
			}
			if !errors.Is(err, iamDomain.ErrGroupNotFound) {
				return err
			}
		}

		g.Name = in.Name
		g.Description = in.Description
		g.UpdatedAt = time.Now()

		if err := uc.groupRepo.Update(ctx, g); err != nil {
			return err
		}
		group = g
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateGroup: %w", err)
	}
	return group, nil
}
