package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ShippingAddresses is the tenant's label sender and return address, plus
// the company profile address the sender falls back to when none is set.
type ShippingAddresses struct {
	Sender         *shippingDomain.ShippingAddress
	Return         *shippingDomain.ShippingAddress
	CompanyAddress *shippingDomain.ShippingAddress
}

type GetShippingAddressesUseCase struct {
	db       *gorm.DB
	prefs    shippingDomain.ShippingPreferencesRepository
	profiles companyProfileGetter
}

func NewGetShippingAddressesUseCase(db *gorm.DB, prefs shippingDomain.ShippingPreferencesRepository, profiles companyProfileGetter) *GetShippingAddressesUseCase {
	return &GetShippingAddressesUseCase{db: db, prefs: prefs, profiles: profiles}
}

func (uc *GetShippingAddressesUseCase) Execute(ctx context.Context) (*ShippingAddresses, error) {
	var out ShippingAddresses
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		prefs, err := uc.prefs.Get(ctx)
		if err != nil {
			return err
		}
		if prefs != nil {
			out.Sender, out.Return = prefs.SenderAddress, prefs.ReturnAddress
		}
		profile, err := uc.profiles.Get(ctx)
		if err != nil {
			return err
		}
		if c := companySender(profile); c != nil {
			out.CompanyAddress = &shippingDomain.ShippingAddress{
				Company: c.Company, Street: c.Street, Number: c.Number, PostalCode: c.PostalCode,
				City: c.City, CC: c.CC, Email: c.Email, Phone: c.Phone,
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getShippingAddresses: %w", err)
	}
	return &out, nil
}

// UpdateShippingAddressesInput replaces both addresses; nil clears one.
type UpdateShippingAddressesInput struct {
	Sender *shippingDomain.ShippingAddress
	Return *shippingDomain.ShippingAddress
}

type UpdateShippingAddressesUseCase struct {
	db     *gorm.DB
	prefs  shippingDomain.ShippingPreferencesRepository
	outbox eventrecord.Enqueuer
}

func NewUpdateShippingAddressesUseCase(db *gorm.DB, prefs shippingDomain.ShippingPreferencesRepository, outbox eventrecord.Enqueuer) *UpdateShippingAddressesUseCase {
	return &UpdateShippingAddressesUseCase{db: db, prefs: prefs, outbox: outbox}
}

func (uc *UpdateShippingAddressesUseCase) Execute(ctx context.Context, in UpdateShippingAddressesInput) error {
	if !tenant.IsTenantAdmin(ctx) {
		return fmt.Errorf("updateShippingAddresses: %w", shippingDomain.ErrShippingPreferencesForbidden)
	}
	sender, err := normalizedAddress(in.Sender)
	if err != nil {
		return fmt.Errorf("updateShippingAddresses: sender: %w", err)
	}
	ret, err := normalizedAddress(in.Return)
	if err != nil {
		return fmt.Errorf("updateShippingAddresses: return: %w", err)
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		before, err := uc.prefs.Get(ctx)
		if err != nil {
			return err
		}
		next := shippingDomain.ShippingPreferences{}
		if before != nil {
			next = *before
		}
		changes := eventrecord.NewChanges().
			Set("sender_address", next.SenderAddress, sender).
			Set("return_address", next.ReturnAddress, ret)
		tenantID := tenant.IDFromContext(ctx)
		next.TenantID, next.SenderAddress, next.ReturnAddress, next.UpdatedAt = tenantID, sender, ret, time.Now()
		if err := uc.prefs.Upsert(ctx, &next); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ShippingAddressesUpdated, tenantID, changes)
	})
	if err != nil {
		return fmt.Errorf("updateShippingAddresses: %w", err)
	}
	return nil
}

func normalizedAddress(a *shippingDomain.ShippingAddress) (*shippingDomain.ShippingAddress, error) {
	if a == nil {
		return nil, nil
	}
	n, err := shippingDomain.NewShippingAddress(*a)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
