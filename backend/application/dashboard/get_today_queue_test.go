package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appdashboard "github.com/adirkuhn/commerceos/application/dashboard"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

// queueKind names which work queue a ListSalesOrdersFilter selects, so the
// stub can answer per queue and the tests can assert the exact filter each
// count uses (the same one the Picking/Packing/Stock issues/Unmatched SKUs
// pages use).
type queueKind string

const (
	kindToPick        queueKind = "to_pick"
	kindToPack        queueKind = "to_pack"
	kindStockIssues   queueKind = "stock_issues"
	kindUnmatchedSKUs queueKind = "unmatched_skus"
	kindShipToday     queueKind = "ship_today"
)

func kindOf(t *testing.T, f omsDomain.ListSalesOrdersFilter) queueKind {
	t.Helper()
	isTrue := func(b *bool) bool { return b != nil && *b }
	if f.ShipDueOnOrBefore != nil {
		if f.AwaitingPicking || f.AwaitingPacking || f.NeedsStockReview != nil || f.NeedsSKUReview != nil {
			t.Fatalf("ship-today filter must stand alone: %+v", f)
		}
		return kindShipToday
	}
	switch {
	case f.AwaitingPicking && !f.AwaitingPacking && f.NeedsStockReview == nil && f.NeedsSKUReview == nil:
		return kindToPick
	case f.AwaitingPacking && !f.AwaitingPicking && f.NeedsStockReview == nil && f.NeedsSKUReview == nil:
		return kindToPack
	case isTrue(f.NeedsStockReview) && !f.AwaitingPicking && !f.AwaitingPacking && f.NeedsSKUReview == nil:
		return kindStockIssues
	case isTrue(f.NeedsSKUReview) && !f.AwaitingPicking && !f.AwaitingPacking && f.NeedsStockReview == nil:
		return kindUnmatchedSKUs
	}
	t.Fatalf("unexpected filter: %+v", f)
	return ""
}

type stubOrderCounter struct {
	t          *testing.T
	counts     map[queueKind]int64
	errFor     queueKind
	gotShipDue *time.Time
}

func (s *stubOrderCounter) Count(_ context.Context, f omsDomain.ListSalesOrdersFilter) (int64, error) {
	k := kindOf(s.t, f)
	if k == kindShipToday {
		s.gotShipDue = f.ShipDueOnOrBefore
	}
	if k == s.errFor {
		return 0, assert.AnError
	}
	return s.counts[k], nil
}

type stubBatchCounter struct {
	expiring       int64
	expiringErr    error
	gotWithinDays  int
	pendingExpired int64
	pendingErr     error
}

func (s *stubBatchCounter) CountExpiringSoon(_ context.Context, withinDays int) (int64, error) {
	s.gotWithinDays = withinDays
	return s.expiring, s.expiringErr
}

func (s *stubBatchCounter) CountPendingReview(_ context.Context) (int64, error) {
	return s.pendingExpired, s.pendingErr
}

func TestGetTodayQueue_composesEveryCount(t *testing.T) {
	orders := &stubOrderCounter{t: t, counts: map[queueKind]int64{
		kindToPick: 4, kindToPack: 7, kindStockIssues: 2, kindUnmatchedSKUs: 3, kindShipToday: 6,
	}}
	batches := &stubBatchCounter{expiring: 5, pendingExpired: 1}
	uc := appdashboard.NewGetTodayQueueUseCase(newMockGormDB(t), orders, batches, &stubSettingsRepo{})

	got, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, appdashboard.TodayQueue{
		ToPick: 4, ToPack: 7, StockIssues: 2, UnmatchedSKUs: 3,
		ExpiringBatches: 5, ExpiredBatches: 1, ShipToday: 6,
	}, *got)
	assert.Equal(t, 7, batches.gotWithinDays, "expiring window is the dashboard's 7 days")
}

func TestGetTodayQueue_wrapsEachOrderCountError(t *testing.T) {
	for _, k := range []queueKind{kindToPick, kindToPack, kindStockIssues, kindUnmatchedSKUs, kindShipToday} {
		t.Run(string(k), func(t *testing.T) {
			orders := &stubOrderCounter{t: t, errFor: k}
			uc := appdashboard.NewGetTodayQueueUseCase(newMockGormDB(t), orders, &stubBatchCounter{}, &stubSettingsRepo{})

			got, err := uc.Execute(context.Background())

			require.ErrorIs(t, err, assert.AnError)
			assert.Contains(t, err.Error(), "getTodayQueue")
			assert.Nil(t, got)
		})
	}
}

func TestGetTodayQueue_wrapsExpiringBatchesError(t *testing.T) {
	uc := appdashboard.NewGetTodayQueueUseCase(newMockGormDB(t),
		&stubOrderCounter{t: t}, &stubBatchCounter{expiringErr: assert.AnError}, &stubSettingsRepo{})

	got, err := uc.Execute(context.Background())

	require.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "getTodayQueue")
	assert.Nil(t, got)
}

func TestGetTodayQueue_wrapsPendingExpiredError(t *testing.T) {
	uc := appdashboard.NewGetTodayQueueUseCase(newMockGormDB(t),
		&stubOrderCounter{t: t}, &stubBatchCounter{pendingErr: assert.AnError}, &stubSettingsRepo{})

	got, err := uc.Execute(context.Background())

	require.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "getTodayQueue")
	assert.Nil(t, got)
}

func TestGetTodayQueue_shipTodayUsesTodayInTheTenantTimezone(t *testing.T) {
	orders := &stubOrderCounter{t: t}
	settings := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{Timezone: "Europe/Amsterdam"}}
	uc := appdashboard.NewGetTodayQueueUseCase(newMockGormDB(t), orders, &stubBatchCounter{}, settings)
	// 22:30 UTC on the 12th is already the 13th in Amsterdam.
	uc.Now = func() time.Time { return time.Date(2026, 10, 12, 22, 30, 0, 0, time.UTC) }

	_, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.NotNil(t, orders.gotShipDue)
	assert.Equal(t, "2026-10-13", orders.gotShipDue.Format(time.DateOnly))
}

func TestGetTodayQueue_wrapsSettingsError(t *testing.T) {
	uc := appdashboard.NewGetTodayQueueUseCase(newMockGormDB(t),
		&stubOrderCounter{t: t}, &stubBatchCounter{}, &stubSettingsRepo{err: assert.AnError})

	got, err := uc.Execute(context.Background())

	require.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "getTodayQueue")
	assert.Nil(t, got)
}
