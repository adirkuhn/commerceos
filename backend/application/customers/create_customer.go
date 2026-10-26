package customers

import (
	"context"
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
	PhoneE164        string
	Username         string
	Role             string
	Billing          customersDomain.Address
	Shipping         customersDomain.Address
	IsPayingCustomer bool
	AvatarURL        string
	CustomerNote     string
}

type CreateCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
	outbox       eventrecord.Enqueuer
}

func NewCreateCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository, outbox eventrecord.Enqueuer) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{db: db, customerRepo: customerRepo, outbox: outbox}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, in CreateCustomerInput) (*customersDomain.Customer, error) {
	probe := customersDomain.Customer{Type: in.Type, CompanyName: in.CompanyName, Email: in.Email, FirstName: in.FirstName, LastName: in.LastName, PhoneE164: in.PhoneE164, Billing: in.Billing}
	if err := probe.Validate(); err != nil {
		return nil, fmt.Errorf("createCustomer: %w", err)
	}
	var customer customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		customer = customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), WooCustomerID: in.WooCustomerID,
			Type: probe.TypeOrPrivate(), CompanyName: in.CompanyName, ClientNumber: in.ClientNumber, KVKNumber: in.KVKNumber, VATNumber: in.VATNumber,
			Email: in.Email, FirstName: in.FirstName, LastName: in.LastName, PhoneE164: in.PhoneE164,
			Username: in.Username, Role: in.Role, Billing: in.Billing, Shipping: in.Shipping,
			IsPayingCustomer: in.IsPayingCustomer, AvatarURL: in.AvatarURL, CustomerNote: in.CustomerNote,
			DateCreated: now, DateModified: now,
		}
		if err := uc.customerRepo.Create(ctx, &customer); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CustomerCreated, customer.ID, eventrecord.Data{
			"woo_customer_id": customer.WooCustomerID, "role": customer.Role, "is_paying_customer": customer.IsPayingCustomer,
			"customer_type": customer.Type, "client_number": customer.ClientNumber,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createCustomer: %w", err)
	}
	return &customer, nil
}
