package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	outbox        eventrecord.Enqueuer
}

func NewUpdateWarehouseUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository, outbox eventrecord.Enqueuer) *UpdateWarehouseUseCase {
	return &UpdateWarehouseUseCase{db: db, warehouseRepo: warehouseRepo, outbox: outbox}
}

func (uc *UpdateWarehouseUseCase) Execute(ctx context.Context, in UpdateWarehouseInput) (*invDomain.Warehouse, error) {
	var warehouse invDomain.Warehouse
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.warehouseRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		before := *existing
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
		return eventrecord.Record(ctx, uc.outbox, events.WarehouseUpdated, warehouse.ID, eventrecord.NewChanges().
			Set("name", before.Name, warehouse.Name).
			Set("address1", before.Address1, warehouse.Address1).
			Set("address2", before.Address2, warehouse.Address2).
			Set("city", before.City, warehouse.City).
			Set("province", before.Province, warehouse.Province).
			Set("zip", before.Zip, warehouse.Zip).
			Set("country", before.Country, warehouse.Country).
			Set("phone", before.Phone, warehouse.Phone).
			Set("active", before.Active, warehouse.Active))
	})
	if err != nil {
		return nil, fmt.Errorf("updateWarehouse: %w", err)
	}
	return &warehouse, nil
}
