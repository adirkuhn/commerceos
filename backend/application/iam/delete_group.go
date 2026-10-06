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

type DeleteGroupUseCase struct {
	db        *gorm.DB
	groupRepo iamDomain.GroupRepository
	outbox    eventrecord.Enqueuer
}

func NewDeleteGroupUseCase(db *gorm.DB, groupRepo iamDomain.GroupRepository, outbox eventrecord.Enqueuer) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{db: db, groupRepo: groupRepo, outbox: outbox}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.groupRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.groupRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.GroupDeleted, id, eventrecord.Data{"name": existing.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteGroup: %w", err)
	}
	return nil
}
