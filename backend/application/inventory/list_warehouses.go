package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListWarehousesUseCase struct {
	db            *gorm.DB
	warehouseRepo invDomain.WarehouseRepository
}

func NewListWarehousesUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository) *ListWarehousesUseCase {
	return &ListWarehousesUseCase{db: db, warehouseRepo: warehouseRepo}
}

func (uc *ListWarehousesUseCase) Execute(ctx context.Context) ([]invDomain.Warehouse, error) {
	var warehouses []invDomain.Warehouse
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		warehouses, err = uc.warehouseRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listWarehouses: %w", err)
	}
	return warehouses, nil
}
