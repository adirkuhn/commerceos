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

type UpdateSupplierInput struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	Country      string
	LeadTimeDays *int
	Status       suppliersDomain.SupplierStatus
	Notes        string
}

type UpdateSupplierUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
	outbox       eventrecord.Enqueuer
}

func NewUpdateSupplierUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository, outbox eventrecord.Enqueuer) *UpdateSupplierUseCase {
	return &UpdateSupplierUseCase{db: db, supplierRepo: supplierRepo, outbox: outbox}
}

func (uc *UpdateSupplierUseCase) Execute(ctx context.Context, in UpdateSupplierInput) (*suppliersDomain.Supplier, error) {
	var supplier suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.supplierRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		before := *existing
		existing.Name = in.Name
		existing.Email = in.Email
		existing.Phone = in.Phone
		existing.Country = in.Country
		existing.LeadTimeDays = in.LeadTimeDays
		existing.Status = in.Status
		existing.Notes = in.Notes
		existing.UpdatedAt = time.Now()
		if err := uc.supplierRepo.Update(ctx, existing); err != nil {
			return err
		}
		supplier = *existing
		return eventrecord.Record(ctx, uc.outbox, events.SupplierUpdated, supplier.ID, eventrecord.NewChanges().
			Set("name", before.Name, supplier.Name).
			Set("email", before.Email, supplier.Email).
			Set("phone", before.Phone, supplier.Phone).
			Set("country", before.Country, supplier.Country).
			Set("lead_time_days", before.LeadTimeDays, supplier.LeadTimeDays).
			Set("status", string(before.Status), string(supplier.Status)).
			Set("notes", before.Notes, supplier.Notes))
	})
	if err != nil {
		return nil, fmt.Errorf("updateSupplier: %w", err)
	}
	return &supplier, nil
}
