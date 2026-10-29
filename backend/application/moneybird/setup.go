package moneybird

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

// Setup is the Moneybird connection's settings plus what they choose from.
// TaxRates and LedgerAccounts are empty until an administration is chosen.
type Setup struct {
	ConnectionID    string
	Settings        mb.Settings
	Administrations []mb.Administration
	TaxRates        []mb.TaxRate
	LedgerAccounts  []mb.LedgerAccount
	PaymentMethods  []string
}

// GetSetupUseCase reads the settings and lists the administrations, tax
// rates and ledger accounts live from Moneybird.
type GetSetupUseCase struct {
	db       *gorm.DB
	conns    connectionLister
	settings mb.SettingsRepository
	factory  mb.ClientFactory
	secret   string
}

func NewGetSetupUseCase(db *gorm.DB, conns connectionLister, settings mb.SettingsRepository, factory mb.ClientFactory, secret string) *GetSetupUseCase {
	return &GetSetupUseCase{db: db, conns: conns, settings: settings, factory: factory, secret: secret}
}

func (uc *GetSetupUseCase) Execute(ctx context.Context) (*Setup, error) {
	acc, err := loadAccount(ctx, uc.db, uc.conns, uc.settings, uc.secret)
	if err != nil {
		return nil, fmt.Errorf("getMoneybirdSetup: %w", err)
	}
	setup := &Setup{ConnectionID: acc.conn.ID, Settings: acc.settings, PaymentMethods: mb.PaymentMethods}
	if setup.Administrations, err = uc.factory.New(acc.token, "").ListAdministrations(ctx); err != nil {
		return nil, fmt.Errorf("getMoneybirdSetup: %w", err)
	}
	if acc.settings.AdministrationID == "" {
		return setup, nil
	}
	client := acc.client(uc.factory)
	if setup.TaxRates, err = client.ListSalesTaxRates(ctx); err != nil {
		return nil, fmt.Errorf("getMoneybirdSetup: %w", err)
	}
	if setup.LedgerAccounts, err = client.ListRevenueLedgerAccounts(ctx); err != nil {
		return nil, fmt.Errorf("getMoneybirdSetup: %w", err)
	}
	return setup, nil
}
