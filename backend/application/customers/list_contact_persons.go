package customers

import (
	"context"
	"fmt"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListContactPersonsUseCase lists a customer's contact persons, oldest
// (the main one) first.
type ListContactPersonsUseCase struct {
	db        *gorm.DB
	customers customerFinder
	persons   customersDomain.ContactPersonRepository
}

func NewListContactPersonsUseCase(db *gorm.DB, customers customerFinder, persons customersDomain.ContactPersonRepository) *ListContactPersonsUseCase {
	return &ListContactPersonsUseCase{db: db, customers: customers, persons: persons}
}

func (uc *ListContactPersonsUseCase) Execute(ctx context.Context, customerID string) ([]customersDomain.ContactPerson, error) {
	var out []customersDomain.ContactPerson
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.customers.FindByID(ctx, customerID); err != nil {
			return err
		}
		var err error
		out, err = uc.persons.ListByCustomer(ctx, customerID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listContactPersons: %w", err)
	}
	return out, nil
}
