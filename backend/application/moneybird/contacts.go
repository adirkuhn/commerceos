package moneybird

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// guestContactName is the shared contact for a sale with nobody to book it
// on (no customer, email or company).
const guestContactName = "Guest customer"

type customerReader interface {
	FindByID(ctx context.Context, id string) (*customersDomain.Customer, error)
}

// party is who a document is for: a CommerceOS customer (remembered against
// their Moneybird contact) and/or the name/address on the document.
type party struct {
	CustomerID string
	Contact    mb.Contact
}

func (p party) anonymous() bool {
	c := p.Contact
	return c.Email == "" && c.CompanyName == "" && c.FirstName == "" && c.LastName == ""
}

// contactResolver finds the Moneybird contact for a party: the one
// remembered for the customer, else one with the same email or company
// name, else a new one. A party with nothing to go by gets the guest
// contact, created on first use and kept in the settings.
type contactResolver struct {
	db       *gorm.DB
	contacts mb.ContactRepository
	settings mb.SettingsRepository
	outbox   eventrecord.Enqueuer
}

func (r contactResolver) resolve(ctx context.Context, acc *account, client mb.Client, p party) (string, error) {
	if p.anonymous() && p.CustomerID == "" {
		return r.guest(ctx, acc, client)
	}
	if p.CustomerID != "" {
		id, err := r.remembered(ctx, acc, client, p.CustomerID)
		if err != nil || id != "" {
			return id, err
		}
	}
	for _, q := range []string{p.Contact.Email, p.Contact.CompanyName} {
		if q == "" {
			continue
		}
		found, err := client.FindContact(ctx, q)
		if err != nil {
			return "", err
		}
		if found != nil {
			return found.ID, r.remember(ctx, acc, p.CustomerID, found.ID)
		}
	}
	if p.anonymous() {
		return r.guest(ctx, acc, client)
	}
	created, err := client.CreateContact(ctx, p.Contact)
	if err != nil {
		return "", err
	}
	return created.ID, r.remember(ctx, acc, p.CustomerID, created.ID)
}

// remembered is the customer's stored contact, when it still exists in
// Moneybird (it may have been deleted there).
func (r contactResolver) remembered(ctx context.Context, acc *account, client mb.Client, customerID string) (string, error) {
	var id string
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		var err error
		id, err = r.contacts.Find(ctx, acc.conn.ID, customerID)
		return err
	})
	if err != nil || id == "" {
		return "", err
	}
	if _, err := client.GetContact(ctx, id); err != nil {
		return "", nil
	}
	return id, nil
}

//events:none moneybird_contacts only caches which Moneybird contact a customer is; the booking it serves raises moneybird_document.synced
func (r contactResolver) remember(ctx context.Context, acc *account, customerID, contactID string) error {
	if customerID == "" {
		return nil
	}
	return tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		return r.contacts.Save(ctx, acc.conn.TenantID, acc.conn.ID, customerID, contactID)
	})
}

func (r contactResolver) guest(ctx context.Context, acc *account, client mb.Client) (string, error) {
	if id := acc.settings.GuestContactID; id != "" {
		if _, err := client.GetContact(ctx, id); err == nil {
			return id, nil
		}
	}
	created, err := client.CreateContact(ctx, mb.Contact{CompanyName: guestContactName})
	if err != nil {
		return "", err
	}
	before := acc.settings.GuestContactID
	acc.settings.GuestContactID, acc.settings.UpdatedAt = created.ID, time.Now()
	err = tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		if err := r.settings.Upsert(ctx, &acc.settings); err != nil {
			return err
		}
		return eventrecord.Record(ctx, r.outbox, events.MoneybirdSettingsUpdated, acc.conn.ID,
			eventrecord.NewChanges().Set("guest_contact_id", before, created.ID))
	})
	return created.ID, err
}

// contactFromCustomer prefers the customer record, falling back to the
// document's own address for what it lacks.
func contactFromCustomer(c *customersDomain.Customer, fallback mb.Contact) mb.Contact {
	if c == nil {
		return fallback
	}
	out := mb.Contact{
		CompanyName: firstNonBlank(c.CompanyName, c.Billing.Company, fallback.CompanyName),
		FirstName:   firstNonBlank(c.FirstName, c.Billing.FirstName, fallback.FirstName),
		LastName:    firstNonBlank(c.LastName, c.Billing.LastName, fallback.LastName),
		Email:       firstNonBlank(c.Email, c.Billing.Email, fallback.Email),
		Address1:    firstNonBlank(c.Billing.Address1, fallback.Address1),
		Address2:    firstNonBlank(c.Billing.Address2, fallback.Address2),
		ZipCode:     firstNonBlank(c.Billing.Postcode, fallback.ZipCode),
		City:        firstNonBlank(c.Billing.City, fallback.City),
		Country:     firstNonBlank(c.Billing.Country, fallback.Country),
	}
	return out
}

func firstNonBlank(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
