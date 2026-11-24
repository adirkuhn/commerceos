package customers

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"time"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateCustomerInput struct {
	ID            string
	WooCustomerID *int64
	// Type/CompanyName/ClientNumber/KVKNumber/VATNumber: nil keeps the
	// current value.
	Type         *customersDomain.Type
	CompanyName  *string
	ClientNumber *string
	KVKNumber    *string
	VATNumber    *string
	Email        string
	FirstName    string
	LastName     string
	Username     string
	Role         string
	Billing      customersDomain.Address
	// Shipping, when sent, becomes the customer's default shipping
	// address (edited, or added when they have none); nil keeps the book.
	// Ignored while shipping to billing.
	Shipping *customersDomain.Address
	// ShipToBilling: nil keeps the current value.
	ShipToBilling *bool
	AvatarURL     string
	CustomerNote  string
}

type UpdateCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
	addresses    customersDomain.ShippingAddressRepository
	outbox       eventrecord.Enqueuer
}

func NewUpdateCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository, addresses customersDomain.ShippingAddressRepository, outbox eventrecord.Enqueuer) *UpdateCustomerUseCase {
	return &UpdateCustomerUseCase{db: db, customerRepo: customerRepo, addresses: addresses, outbox: outbox}
}

func (uc *UpdateCustomerUseCase) Execute(ctx context.Context, in UpdateCustomerInput) (*customersDomain.Customer, error) {
	var customer customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.customerRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		before := *existing
		existing.WooCustomerID = in.WooCustomerID
		existing.Email = in.Email
		existing.FirstName = in.FirstName
		existing.LastName = in.LastName
		existing.Username = in.Username
		existing.Role = in.Role
		existing.Billing = in.Billing
		if in.ShipToBilling != nil {
			existing.ShipToBilling = *in.ShipToBilling
		}
		existing.AvatarURL = in.AvatarURL
		existing.CustomerNote = in.CustomerNote
		applyCompanyFields(existing, in)
		existing.DateModified = time.Now()
		if err := existing.Validate(); err != nil {
			return err
		}
		if err := uc.applyShipping(ctx, existing, in.Shipping); err != nil {
			return err
		}
		if err := uc.customerRepo.Update(ctx, existing); err != nil {
			return err
		}
		customer = *existing
		return recordCustomerUpdated(ctx, uc.outbox, before, customer)
	})
	if err != nil {
		return nil, fmt.Errorf("updateCustomer: %w", err)
	}
	return &customer, nil
}

func applyCompanyFields(c *customersDomain.Customer, in UpdateCustomerInput) {
	if in.Type != nil {
		c.Type = *in.Type
	}
	setIfSent(&c.CompanyName, in.CompanyName)
	setIfSent(&c.ClientNumber, in.ClientNumber)
	setIfSent(&c.KVKNumber, in.KVKNumber)
	setIfSent(&c.VATNumber, in.VATNumber)
}

func setIfSent(field *string, value *string) {
	if value != nil {
		*field = *value
	}
}

// applyShipping saves a sent shipping address as the default one (unless
// shipping to billing) and points c.Shipping at where goods now go.
func (uc *UpdateCustomerUseCase) applyShipping(ctx context.Context, c *customersDomain.Customer, sent *customersDomain.Address) error {
	book, err := uc.addresses.ListByCustomer(ctx, c.ID)
	if err != nil {
		return err
	}
	if sent != nil && !c.ShipToBilling {
		if book, err = uc.saveDefault(ctx, c, book, *sent); err != nil {
			return err
		}
	}
	c.Shipping = c.EffectiveShipping(book.Default())
	return nil
}

// saveDefault edits the default address to sent, or adds sent as the
// first default -- through the same use cases staff use, so the rules and
// events are the same. Returns the book as it now is.
func (uc *UpdateCustomerUseCase) saveDefault(ctx context.Context, c *customersDomain.Customer, book customersDomain.ShippingBook, sent customersDomain.Address) (customersDomain.ShippingBook, error) {
	sent.Email = ""
	def := book.Default()
	switch {
	case def != nil && def.Address == sent:
		return book, nil
	case def != nil:
		edited := *def
		in := ShippingAddressInput{Label: def.Label, Address: sent}
		in.applyTo(&edited)
		if err := edited.Validate(); err != nil {
			return nil, err
		}
		edited.UpdatedAt = time.Now()
		if err := uc.addresses.Update(ctx, &edited); err != nil {
			return nil, err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.CustomerShippingAddressUpdated, def.ID,
			eventrecord.Data{"customer_id": c.ID, "changed_fields": changedShippingFields(*def, edited)}); err != nil {
			return nil, err
		}
	case sent.IsEmpty():
		return book, nil
	default:
		first, err := book.Add(customersDomain.ShippingAddress{Address: sent}, true)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		first.TenantID, first.CustomerID, first.CreatedAt, first.UpdatedAt = c.TenantID, c.ID, now, now
		if err := uc.addresses.Create(ctx, &first); err != nil {
			return nil, err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.CustomerShippingAddressAdded, first.ID,
			eventrecord.Data{"customer_id": c.ID, "default": true}); err != nil {
			return nil, err
		}
	}
	return uc.addresses.ListByCustomer(ctx, c.ID)
}
