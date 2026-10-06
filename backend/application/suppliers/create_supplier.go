package suppliers

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateSupplierInput struct {
	Name         string
	Email        string
	Phone        string
	Country      string
	LeadTimeDays *int
	Status       suppliersDomain.SupplierStatus
	Notes        string
}

type CreateSupplierUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
	outbox       eventrecord.Enqueuer
}

func NewCreateSupplierUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository, outbox eventrecord.Enqueuer) *CreateSupplierUseCase {
	return &CreateSupplierUseCase{db: db, supplierRepo: supplierRepo, outbox: outbox}
}

func (uc *CreateSupplierUseCase) Execute(ctx context.Context, in CreateSupplierInput) (*suppliersDomain.Supplier, error) {
	var supplier suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		supplier = suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx),
			Name:     in.Name, Email: in.Email, Phone: in.Phone, Country: in.Country,
			LeadTimeDays: in.LeadTimeDays, Status: in.Status, Notes: in.Notes,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.supplierRepo.Create(ctx, &supplier); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SupplierCreated, supplier.ID,
			eventrecord.Data{"name": supplier.Name, "status": string(supplier.Status)})
	})
	if err != nil {
		return nil, fmt.Errorf("createSupplier: %w", err)
	}
	return &supplier, nil
}
