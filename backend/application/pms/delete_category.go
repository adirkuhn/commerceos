package pms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteCategoryUseCase struct {
	db           *gorm.DB
	categoryRepo pmsDomain.CategoryRepository
	outbox       eventrecord.Enqueuer
}

func NewDeleteCategoryUseCase(db *gorm.DB, categoryRepo pmsDomain.CategoryRepository, outbox eventrecord.Enqueuer) *DeleteCategoryUseCase {
	return &DeleteCategoryUseCase{db: db, categoryRepo: categoryRepo, outbox: outbox}
}

func (uc *DeleteCategoryUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		category, err := uc.categoryRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.categoryRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CategoryDeleted, id, eventrecord.Data{"name": category.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteCategory: %w", err)
	}
	return nil
}
