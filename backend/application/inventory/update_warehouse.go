package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateWarehouseInput struct {
	ID       string
	Name     string
	Address1 string
	Address2 string
	City     string
	Province string
	Zip      string
	Country  string
	Phone    string
	Active   bool
}

type UpdateWarehouseUseCase struct {
	db            *gorm.DB
	warehouseRepo invDomain.WarehouseRepository
}

func NewUpdateWarehouseUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository) *UpdateWarehouseUseCase {
	return &UpdateWarehouseUseCase{db: db, warehouseRepo: warehouseRepo}
}

func (uc *UpdateWarehouseUseCase) Execute(ctx context.Context, in UpdateWarehouseInput) (*invDomain.Warehouse, error) {
	var warehouse invDomain.Warehouse
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.warehouseRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Name = in.Name
		existing.Address1 = in.Address1
		existing.Address2 = in.Address2
		existing.City = in.City
		existing.Province = in.Province
		existing.Zip = in.Zip
		existing.Country = in.Country
		existing.Phone = in.Phone
		existing.Active = in.Active
		existing.UpdatedAt = time.Now()
		if err := uc.warehouseRepo.Update(ctx, existing); err != nil {
			return err
		}
		warehouse = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateWarehouse: %w", err)
	}
	return &warehouse, nil
}
