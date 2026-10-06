package iam

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"time"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateGroupInput struct {
	Name        string
	Description string
}

type CreateGroupUseCase struct {
	db        *gorm.DB
	groupRepo iamDomain.GroupRepository
	outbox    eventrecord.Enqueuer
}

func NewCreateGroupUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository, outbox eventrecord.Enqueuer) *CreateGroupUseCase {
	return &CreateGroupUseCase{db: db, groupRepo: groupRepo, outbox: outbox}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, in CreateGroupInput) (*iamDomain.Group, error) {
	var group iamDomain.Group
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		_, err := uc.groupRepo.FindByName(ctx, in.Name)
		if err == nil {
			return iamDomain.ErrGroupNameTaken
		}
		if !errors.Is(err, iamDomain.ErrGroupNotFound) {
			return err
		}

		now := time.Now()
		group = iamDomain.Group{
			TenantID:    tenant.IDFromContext(ctx),
			Name:        in.Name,
			Description: in.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := uc.groupRepo.Create(ctx, &group); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.GroupCreated, group.ID, eventrecord.Data{
			"name": group.Name, "description": group.Description,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createGroup: %w", err)
	}
	return &group, nil
}
