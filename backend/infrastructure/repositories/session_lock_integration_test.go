//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// locked_since must survive heartbeats from the same holder, move when the
// holder changes, and clear on release -- the packing queue shows it as
// "how long has this person had the order".
func TestPickingSessionRepo_Integration_LockedSinceTracksHolderNotHeartbeat(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewPickingSessionRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	var sessionID string
	inTx := func(fn func(ctx context.Context) error) { require.NoError(t, tenant.WithTenant(ctx, db, fn)) }
	find := func() *omsDomain.PickingSession {
		var s *omsDomain.PickingSession
		inTx(func(ctx context.Context) error {
			var err error
			s, err = repo.FindByID(ctx, sessionID)
			return err
		})
		return s
	}
	inTx(func(ctx context.Context) error {
		s := omsDomain.PickingSession{TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now()}
		err := repo.Create(ctx, &s)
		sessionID = s.ID
		return err
	})

	t0 := time.Now().Add(-10 * time.Minute).Truncate(time.Microsecond)
	t1 := t0.Add(time.Minute)
	t2 := t1.Add(time.Minute)

	inTx(func(ctx context.Context) error { return repo.AcquireLock(ctx, sessionID, "u1", t0) })
	require.True(t, find().LockedSince.Equal(t0))

	inTx(func(ctx context.Context) error { return repo.AcquireLock(ctx, sessionID, "u1", t1) })
	got := find()
	assert.True(t, got.LockedAt.Equal(t1), "heartbeat moves locked_at")
	assert.True(t, got.LockedSince.Equal(t0), "heartbeat keeps locked_since")

	inTx(func(ctx context.Context) error { return repo.AcquireLock(ctx, sessionID, "u2", t2) })
	assert.True(t, find().LockedSince.Equal(t2), "new holder restarts locked_since")

	inTx(func(ctx context.Context) error { return repo.ReleaseLock(ctx, sessionID) })
	assert.Nil(t, find().LockedSince)
}

func TestPickingSessionRepo_Integration_ListOpenBySalesOrdersAndAbandonedStatus(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewPickingSessionRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	var openID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, st := range []omsDomain.PickingSessionStatus{omsDomain.PickingSessionStatusAbandoned, omsDomain.PickingSessionStatusCompleted, omsDomain.PickingSessionStatusPaused} {
			s := omsDomain.PickingSession{TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Status: st, StartedAt: time.Now()}
			if err := repo.Create(ctx, &s); err != nil {
				return err
			}
			if st == omsDomain.PickingSessionStatusPaused {
				openID = s.ID
			}
		}
		return nil
	})
	require.NoError(t, err, "abandoned must satisfy the status check constraint")

	var open []omsDomain.PickingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		open, err = repo.ListOpenBySalesOrders(ctx, []string{orderID, "00000000-0000-0000-0000-000000000000"})
		return err
	})
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, openID, open[0].ID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		open, err = repo.ListOpenBySalesOrders(ctx, nil)
		return err
	})
	require.NoError(t, err)
	assert.Empty(t, open)
}

func TestPackingSessionRepo_Integration_ListActiveBySalesOrdersAndLockedSince(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewPackingSessionRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID, _ := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	var activeID string
	at := time.Now().Truncate(time.Microsecond)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, st := range []omsDomain.PackingSessionStatus{omsDomain.PackingSessionStatusCompleted, omsDomain.PackingSessionStatusInProgress} {
			s := omsDomain.PackingSession{TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Status: st, StartedAt: time.Now()}
			if err := repo.Create(ctx, &s); err != nil {
				return err
			}
			activeID = s.ID
		}
		return repo.AcquireLock(ctx, activeID, "u1", at)
	})
	require.NoError(t, err)

	var active []omsDomain.PackingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		active, err = repo.ListActiveBySalesOrders(ctx, []string{orderID})
		return err
	})
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, activeID, active[0].ID)
	require.NotNil(t, active[0].LockedSince)
	assert.True(t, active[0].LockedSince.Equal(at))
}
