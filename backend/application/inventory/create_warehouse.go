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

type CreateWarehouseInput struct {
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

type CreateWarehouseUseCase struct {
	db            *gorm.DB
	warehouseRepo invDomain.WarehouseRepository
	outbox        eventrecord.Enqueuer
}

func NewCreateWarehouseUseCase(db *gorm.DB, warehouseRepo invDomain.WarehouseRepository, outbox eventrecord.Enqueuer) *CreateWarehouseUseCase {
	return &CreateWarehouseUseCase{db: db, warehouseRepo: warehouseRepo, outbox: outbox}
}

func (uc *CreateWarehouseUseCase) Execute(ctx context.Context, in CreateWarehouseInput) (*invDomain.Warehouse, error) {
	var warehouse invDomain.Warehouse
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		warehouse = invDomain.Warehouse{
			TenantID:  tenant.IDFromContext(ctx),
			Name:      in.Name,
			Address1:  in.Address1,
			Address2:  in.Address2,
			City:      in.City,
			Province:  in.Province,
			Zip:       in.Zip,
			Country:   in.Country,
			Phone:     in.Phone,
			Active:    in.Active,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.warehouseRepo.Create(ctx, &warehouse); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.WarehouseCreated, warehouse.ID, eventrecord.Data{
			"name": warehouse.Name, "country": warehouse.Country, "active": warehouse.Active,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createWarehouse: %w", err)
	}
	return &warehouse, nil
}
