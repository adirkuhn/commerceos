package moneybird

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

type documentSender interface {
	Execute(ctx context.Context, sourceID string) (*mb.Document, error)
}

// AutoSyncUseCase sends quotes and orders as they're created or change,
// when the shop turned auto sync on (off by default: orders may already be
// booked by another service). Does nothing when Moneybird isn't connected
// or set up.
//
// Only a transient failure (Moneybird unreachable or busy) is returned, so
// the event is retried. Any other failure is already recorded on the
// quote's/order's Moneybird status and retrying wouldn't change it.
type AutoSyncUseCase struct {
	db       *gorm.DB
	conns    connectionLister
	settings mb.SettingsRepository
	orders   documentSender
	quotes   documentSender
	secret   string
}

func NewAutoSyncUseCase(db *gorm.DB, conns connectionLister, settings mb.SettingsRepository, orders, quotes documentSender, secret string) *AutoSyncUseCase {
	return &AutoSyncUseCase{db: db, conns: conns, settings: settings, orders: orders, quotes: quotes, secret: secret}
}

func (uc *AutoSyncUseCase) SalesOrder(ctx context.Context, orderID string) error {
	return uc.send(ctx, uc.orders, orderID)
}

func (uc *AutoSyncUseCase) Quote(ctx context.Context, quoteID string) error {
	return uc.send(ctx, uc.quotes, quoteID)
}

func (uc *AutoSyncUseCase) send(ctx context.Context, sender documentSender, id string) error {
	acc, err := loadAccount(ctx, uc.db, uc.conns, uc.settings, uc.secret)
	if errors.Is(err, mb.ErrNotConnected) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("moneybirdAutoSync: %w", err)
	}
	if !acc.settings.AutoSync || acc.settings.Ready() != nil {
		return nil
	}
	_, err = sender.Execute(ctx, id)
	if errors.Is(err, mb.ErrRequestFailed) || errors.Is(err, mb.ErrRateLimited) {
		return fmt.Errorf("moneybirdAutoSync: %w", err)
	}
	return nil
}
