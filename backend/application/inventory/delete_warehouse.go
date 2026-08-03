package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteWarehouseUseCase struct {
	db            *gorm.DB
	warehouseRepo invDomain.WarehouseRepository
}

func NewDeleteWarehouseUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository) *DeleteWarehouseUseCase {
	return &DeleteWarehouseUseCase{db: db, warehouseRepo: warehouseRepo}
}

func (uc *DeleteWarehouseUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.warehouseRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.warehouseRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteWarehouse: %w", err)
	}
	return nil
}
