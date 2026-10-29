package moneybird_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

type recordingSender struct {
	ids []string
	err error
}

func (s *recordingSender) Execute(_ context.Context, id string) (*mb.Document, error) {
	s.ids = append(s.ids, id)
	return nil, s.err
}

func autoSync(t *testing.T, s mb.Settings, conns *fakeConnections, orders, quotes *recordingSender) *appmb.AutoSyncUseCase {
	return appmb.NewAutoSyncUseCase(mockDB(t, 2), conns, newFakeSettings(s), orders, quotes, testSecret)
}

func autoOn() mb.Settings {
	s := configuredSettings()
	s.AutoSync = true
	return s
}

func TestAutoSync_sendsWhenOn(t *testing.T) {
	orders, quotes := &recordingSender{}, &recordingSender{}
	uc := autoSync(t, autoOn(), connected(t, "tok"), orders, quotes)

	require.NoError(t, uc.SalesOrder(tenantCtx(), "so-1"))
	require.NoError(t, uc.Quote(tenantCtx(), "q-1"))

	assert.Equal(t, []string{"so-1"}, orders.ids)
	assert.Equal(t, []string{"q-1"}, quotes.ids)
}

// Off by default: orders may be booked by another service.
func TestAutoSync_offDoesNothing(t *testing.T) {
	orders, quotes := &recordingSender{}, &recordingSender{}
	uc := autoSync(t, configuredSettings(), connected(t, "tok"), orders, quotes)

	require.NoError(t, uc.SalesOrder(tenantCtx(), "so-1"))
	require.NoError(t, uc.Quote(tenantCtx(), "q-1"))

	assert.Empty(t, orders.ids)
	assert.Empty(t, quotes.ids)
}

func TestAutoSync_notConnectedOrNotConfiguredDoesNothing(t *testing.T) {
	orders := &recordingSender{}
	require.NoError(t, autoSync(t, autoOn(), &fakeConnections{}, orders, &recordingSender{}).SalesOrder(tenantCtx(), "so-1"))

	s := autoOn()
	s.DefaultLedgerAccountID = ""
	require.NoError(t, autoSync(t, s, connected(t, "tok"), orders, &recordingSender{}).SalesOrder(tenantCtx(), "so-1"))

	assert.Empty(t, orders.ids)
}

// The failure is on the order's page already; retrying wouldn't change it.
func TestAutoSync_businessFailuresAreNotRetried(t *testing.T) {
	for _, err := range []error{mb.ErrTaxRateNotMapped, mb.ErrDuplicateReference, mb.ErrPaymentMethodNotMapped, mb.ErrTotalsDontAddUp, mb.ErrDocumentLocked} {
		orders := &recordingSender{err: fmt.Errorf("send: %w", err)}
		assert.NoError(t, autoSync(t, autoOn(), connected(t, "tok"), orders, &recordingSender{}).SalesOrder(tenantCtx(), "so-1"), err.Error())
	}
}

// Moneybird unreachable or busy: worth trying again later.
func TestAutoSync_transientFailuresAreReturnedForRetry(t *testing.T) {
	for _, err := range []error{mb.ErrRequestFailed, mb.ErrRateLimited} {
		quotes := &recordingSender{err: fmt.Errorf("send: %w", err)}
		got := autoSync(t, autoOn(), connected(t, "tok"), &recordingSender{}, quotes).Quote(tenantCtx(), "q-1")
		assert.True(t, errors.Is(got, err))
	}
}
