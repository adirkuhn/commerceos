package customers

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"time"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateCustomerInput struct {
	WooCustomerID *int64
	// Type defaults to private. A company needs CompanyName; ClientNumber
	// is generated when empty.
	Type         customersDomain.Type
	CompanyName  string
	ClientNumber string
	KVKNumber    string
	VATNumber    string
	Email        string
	FirstName    string
	LastName     string
	// PhoneE164 is optional -- set by application/whatsapp's new-customer
	// capture flow (the inbound sender's own number), blank for every
	// other caller today.
	PhoneE164    string
	Username     string
	Role         string
	Billing      customersDomain.Address
	Shipping     customersDomain.Address
	AvatarURL    string
	CustomerNote string
	// ContactPersons are added with the customer, in the same
	// transaction; the first is the main one. Needs WithContactPersons.
	ContactPersons []ContactPersonInput
}

type CreateCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
	persons      customersDomain.ContactPersonRepository
	outbox       eventrecord.Enqueuer
}

// WithContactPersons lets the use case create contact persons with the
// customer (CreateCustomerInput.ContactPersons).
func (uc *CreateCustomerUseCase) WithContactPersons(persons customersDomain.ContactPersonRepository) *CreateCustomerUseCase {
	uc.persons = persons
	return uc
}

func NewCreateCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository, outbox eventrecord.Enqueuer) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{db: db, customerRepo: customerRepo, outbox: outbox}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, in CreateCustomerInput) (*customersDomain.Customer, error) {
	probe := customersDomain.Customer{Type: in.Type, CompanyName: in.CompanyName, Email: in.Email, FirstName: in.FirstName, LastName: in.LastName, PhoneE164: in.PhoneE164, Billing: in.Billing}
	if err := probe.Validate(); err != nil {
		return nil, fmt.Errorf("createCustomer: %w", err)
	}
	persons, err := uc.contactPersons(in.ContactPersons)
	if err != nil {
		return nil, fmt.Errorf("createCustomer: %w", err)
	}
	var customer customersDomain.Customer
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		customer = customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), WooCustomerID: in.WooCustomerID,
			Type: probe.TypeOrPrivate(), CompanyName: in.CompanyName, ClientNumber: in.ClientNumber, KVKNumber: in.KVKNumber, VATNumber: in.VATNumber,
			Email: in.Email, FirstName: in.FirstName, LastName: in.LastName, PhoneE164: in.PhoneE164,
			Username: in.Username, Role: in.Role, Billing: in.Billing, Shipping: in.Shipping,
			AvatarURL: in.AvatarURL, CustomerNote: in.CustomerNote,
			DateCreated: now, DateModified: now,
		}
		if err := uc.customerRepo.Create(ctx, &customer); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.CustomerCreated, customer.ID, eventrecord.Data{
			"woo_customer_id": customer.WooCustomerID, "role": customer.Role,
			"customer_type": customer.Type, "client_number": customer.ClientNumber,
		}); err != nil {
			return err
		}
		for i := range persons {
			if err := addContactPerson(ctx, uc.persons, uc.outbox, customer.ID, &persons[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("createCustomer: %w", err)
	}
	return &customer, nil
}

// contactPersons validates the contact persons to create with a customer.
func (uc *CreateCustomerUseCase) contactPersons(in []ContactPersonInput) ([]customersDomain.ContactPerson, error) {
	if len(in) > 0 && uc.persons == nil {
		return nil, errors.New("contact persons given but no contact person repository wired")
	}
	out := make([]customersDomain.ContactPerson, len(in))
	for i, p := range in {
		p.applyTo(&out[i])
		if err := out[i].Validate(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
