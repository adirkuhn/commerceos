package moneybird

import (
	"context"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type settingsConnections interface {
	connectionLister
}

// UpdateSettingsInput: nil leaves a field as it is -- each settings
// section sends only its own fields.
type UpdateSettingsInput struct {
	AdministrationID       *string
	DefaultLedgerAccountID *string
	AutoSync               *bool
	TaxRates               *[]mb.TaxRateMapping
	PaymentMethods         *[]mb.PaymentMethodMapping
}

// UpdateSettingsUseCase changes the Moneybird settings. Every id is
// checked against Moneybird. Switching administration clears what belongs
// to the old one (ledger account, tax rate mappings, guest contact).
type UpdateSettingsUseCase struct {
	db       *gorm.DB
	conns    settingsConnections
	settings mb.SettingsRepository
	factory  mb.ClientFactory
	outbox   eventrecord.Enqueuer
	secret   string
}

func NewUpdateSettingsUseCase(db *gorm.DB, conns settingsConnections, settings mb.SettingsRepository, factory mb.ClientFactory, outbox eventrecord.Enqueuer, secret string) *UpdateSettingsUseCase {
	return &UpdateSettingsUseCase{db: db, conns: conns, settings: settings, factory: factory, outbox: outbox, secret: secret}
}

func (uc *UpdateSettingsUseCase) Execute(ctx context.Context, in UpdateSettingsInput) (*mb.Settings, error) {
	acc, err := loadAccount(ctx, uc.db, uc.conns, uc.settings, uc.secret)
	if err != nil {
		return nil, fmt.Errorf("updateMoneybirdSettings: %w", err)
	}
	before := acc.settings
	after, err := uc.apply(ctx, acc, in)
	if err != nil {
		return nil, fmt.Errorf("updateMoneybirdSettings: %w", err)
	}
	after.UpdatedAt = time.Now()

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.settings.Upsert(ctx, &after); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.MoneybirdSettingsUpdated, after.ConnectionID, eventrecord.NewChanges().
			Set("administration_id", before.AdministrationID, after.AdministrationID).
			Set("default_ledger_account_id", before.DefaultLedgerAccountID, after.DefaultLedgerAccountID).
			Set("auto_sync", before.AutoSync, after.AutoSync).
			Set("guest_contact_id", before.GuestContactID, after.GuestContactID).
			Set("tax_rates", before.TaxRates, after.TaxRates).
			Set("payment_methods", before.PaymentMethods, after.PaymentMethods))
	})
	if err != nil {
		return nil, fmt.Errorf("updateMoneybirdSettings: %w", err)
	}
	return &after, nil
}

func (uc *UpdateSettingsUseCase) apply(ctx context.Context, acc *account, in UpdateSettingsInput) (mb.Settings, error) {
	s := acc.settings
	if in.AdministrationID != nil && *in.AdministrationID != s.AdministrationID {
		admins, err := uc.factory.New(acc.token, "").ListAdministrations(ctx)
		if err != nil {
			return s, err
		}
		i := slices.IndexFunc(admins, func(a mb.Administration) bool { return a.ID == *in.AdministrationID })
		if i < 0 {
			return s, fmt.Errorf("%w: administration %s", ErrUnknownMoneybirdID, *in.AdministrationID)
		}
		s.AdministrationID, s.AdministrationName = admins[i].ID, admins[i].Name
		s.DefaultLedgerAccountID, s.TaxRates, s.GuestContactID = "", nil, ""
	}
	if in.AutoSync != nil {
		s.AutoSync = *in.AutoSync
	}
	if in.PaymentMethods != nil {
		if err := checkPaymentMethods(*in.PaymentMethods); err != nil {
			return s, err
		}
		s.PaymentMethods = *in.PaymentMethods
	}
	if in.DefaultLedgerAccountID == nil && in.TaxRates == nil {
		return s, nil
	}
	if s.AdministrationID == "" {
		return s, mb.ErrNotConfigured
	}
	client := uc.factory.New(acc.token, s.AdministrationID)
	if in.DefaultLedgerAccountID != nil {
		ledgers, err := client.ListRevenueLedgerAccounts(ctx)
		if err != nil {
			return s, err
		}
		if !slices.ContainsFunc(ledgers, func(l mb.LedgerAccount) bool { return l.ID == *in.DefaultLedgerAccountID }) {
			return s, fmt.Errorf("%w: ledger account %s", ErrUnknownMoneybirdID, *in.DefaultLedgerAccountID)
		}
		s.DefaultLedgerAccountID = *in.DefaultLedgerAccountID
	}
	if in.TaxRates != nil {
		rates, err := client.ListSalesTaxRates(ctx)
		if err != nil {
			return s, err
		}
		if err := checkTaxRates(*in.TaxRates, rates); err != nil {
			return s, err
		}
		s.TaxRates = *in.TaxRates
	}
	return s, nil
}

func checkTaxRates(mappings []mb.TaxRateMapping, rates []mb.TaxRate) error {
	seen := map[int32]bool{}
	for _, m := range mappings {
		if seen[m.RateBasisPoints] {
			return fmt.Errorf("%w: VAT rate %d", ErrDuplicateMapping, m.RateBasisPoints)
		}
		seen[m.RateBasisPoints] = true
		if !slices.ContainsFunc(rates, func(r mb.TaxRate) bool { return r.ID == m.MoneybirdTaxRateID }) {
			return fmt.Errorf("%w: tax rate %s", ErrUnknownMoneybirdID, m.MoneybirdTaxRateID)
		}
	}
	return nil
}

func checkPaymentMethods(mappings []mb.PaymentMethodMapping) error {
	seen := map[string]bool{}
	for _, m := range mappings {
		if seen[m.Method] {
			return fmt.Errorf("%w: payment method %s", ErrDuplicateMapping, m.Method)
		}
		seen[m.Method] = true
		if !slices.Contains(mb.PaymentMethods, m.MoneybirdPaymentMethod) {
			return fmt.Errorf("%w: payment method %s", ErrUnknownMoneybirdID, m.MoneybirdPaymentMethod)
		}
	}
	return nil
}
