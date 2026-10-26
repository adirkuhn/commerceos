package customers

import (
	"context"
	"strings"
	"time"
)

// ContactPerson is someone to deal with at a company customer -- the buyer,
// the accounts department. A customer has any number of them.
type ContactPerson struct {
	ID         string
	TenantID   string
	CustomerID string
	FirstName  string
	LastName   string
	// Role is free text, e.g. "Purchasing" or "Employee".
	Role      string
	Email     string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Name is the person's full name.
func (p ContactPerson) Name() string {
	return strings.TrimSpace(strings.TrimSpace(p.FirstName) + " " + strings.TrimSpace(p.LastName))
}

// Validate requires a name; email and phone are optional.
func (p ContactPerson) Validate() error {
	if p.Name() == "" {
		return ErrContactPersonNameRequired
	}
	return nil
}

type ContactPersonRepository interface {
	Create(ctx context.Context, p *ContactPerson) error
	// FindByID returns ErrContactPersonNotFound when the person isn't
	// one of customerID's.
	FindByID(ctx context.Context, customerID, id string) (*ContactPerson, error)
	// ListByCustomer lists a customer's contact persons, oldest first.
	ListByCustomer(ctx context.Context, customerID string) ([]ContactPerson, error)
	Update(ctx context.Context, p *ContactPerson) error
	Delete(ctx context.Context, customerID, id string) error
}
