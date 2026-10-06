package suppliers

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteSupplierUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
	outbox       eventrecord.Enqueuer
}

func NewDeleteSupplierUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository, outbox eventrecord.Enqueuer) *DeleteSupplierUseCase {
	return &DeleteSupplierUseCase{db: db, supplierRepo: supplierRepo, outbox: outbox}
}

func (uc *DeleteSupplierUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.supplierRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.supplierRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SupplierDeleted, id, eventrecord.Data{"name": existing.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteSupplier: %w", err)
	}
	return nil
}
