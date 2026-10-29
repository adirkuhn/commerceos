// Package moneybird books quotes and sales orders in Moneybird (see
// domain/moneybird) and manages the Moneybird connection's settings.
package moneybird

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/errs"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

var (
	// ErrUnknownMoneybirdID: a settings value names an administration,
	// ledger account, tax rate or payment method Moneybird doesn't have.
	ErrUnknownMoneybirdID = errs.NewWithCause("MONEYBIRD_UNKNOWN_ID", "Moneybird doesn't have this administration, ledger account, tax rate or payment method")
	ErrDuplicateMapping   = errs.NewWithCause("MONEYBIRD_DUPLICATE_MAPPING", "each VAT rate and payment method can be mapped only once")
)

type connectionLister interface {
	ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error)
}

// account is the shop's Moneybird connection with its decrypted token and
// settings -- one per shop.
type account struct {
	conn     integrationsDomain.Connection
	token    string
	settings mb.Settings
}

func findMoneybirdConnection(conns []integrationsDomain.Connection) *integrationsDomain.Connection {
	for i := range conns {
		if conns[i].Provider == integrationsDomain.ProviderMoneybird {
			return &conns[i]
		}
	}
	return nil
}

// loadAccount reads the connection and settings in one transaction. A
// connection without a settings row yet gets empty settings.
func loadAccount(ctx context.Context, db *gorm.DB, conns connectionLister, settings mb.SettingsRepository, secret string) (*account, error) {
	var acc account
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		all, err := conns.ListByTenant(ctx)
		if err != nil {
			return err
		}
		conn := findMoneybirdConnection(all)
		if conn == nil {
			return mb.ErrNotConnected
		}
		acc.conn = *conn
		s, err := settings.Get(ctx, conn.ID)
		if err != nil {
			return err
		}
		if s == nil {
			now := time.Now()
			s = &mb.Settings{ConnectionID: conn.ID, TenantID: conn.TenantID, CreatedAt: now, UpdatedAt: now}
		}
		acc.settings = *s
		return nil
	})
	if err != nil {
		return nil, err
	}
	acc.token, err = crypto.Decrypt(secret, acc.conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypting Moneybird token: %w", err)
	}
	return &acc, nil
}

func (a *account) client(f mb.ClientFactory) mb.Client {
	return f.New(a.token, a.settings.AdministrationID)
}
