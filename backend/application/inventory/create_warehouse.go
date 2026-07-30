package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateWarehouseInput struct {
	Name string
}

type CreateWarehouseUseCase struct {
	db            *gorm.DB
	warehouseRepo invDomain.WarehouseRepository
}

func NewCreateWarehouseUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository) *CreateWarehouseUseCase {
	return &CreateWarehouseUseCase{db: db, warehouseRepo: warehouseRepo}
}

func (uc *CreateWarehouseUseCase) Execute(ctx context.Context, in CreateWarehouseInput) (*invDomain.Warehouse, error) {
	var warehouse invDomain.Warehouse
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		warehouse = invDomain.Warehouse{
			TenantID:  tenant.IDFromContext(ctx),
			Name:      in.Name,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return uc.warehouseRepo.Create(ctx, &warehouse)
	})
	if err != nil {
		return nil, fmt.Errorf("createWarehouse: %w", err)
	}
	return &warehouse, nil
}
