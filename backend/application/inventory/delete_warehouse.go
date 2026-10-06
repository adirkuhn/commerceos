package inventory

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteWarehouseUseCase struct {
	db            *gorm.DB
	warehouseRepo invDomain.WarehouseRepository
	outbox        eventrecord.Enqueuer
}

func NewDeleteWarehouseUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository, outbox eventrecord.Enqueuer) *DeleteWarehouseUseCase {
	return &DeleteWarehouseUseCase{db: db, warehouseRepo: warehouseRepo, outbox: outbox}
}

func (uc *DeleteWarehouseUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.warehouseRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.warehouseRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.WarehouseDeleted, id, eventrecord.Data{"name": existing.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteWarehouse: %w", err)
	}
	return nil
}
