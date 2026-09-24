//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedSalesOrderWithLineItemForPackingDB seeds an order plus one real line
// item -- a plain pack scan always targets a real line (only a freebie/new
// item has none, see PackScan's doc comment), so a
// packing test always needs a real line item row to reference.
func seedSalesOrderWithLineItemForPackingDB(t *testing.T, db *gorm.DB, ctx context.Context, skuID, warehouseID string) (orderID, lineItemID string) {
	t.Helper()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-packing-" + tenant.IDFromContext(ctx), WarehouseID: warehouseID,
			CurrencyCode: "USD", Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true,
			TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID

		line := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: &skuID,
			Name: "Widget A", SKUCode: "WID-A", Barcode: "0123456789012", Quantity: 2,
			PriceMinorUnits: 500, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &line); err != nil {
			return err
		}
		lineItemID = line.ID
		return nil
	})
	require.NoError(t, err)
	return orderID, lineItemID
}

func TestPackingSessionRepo_Integration_CreateFindActiveUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPackingSessionRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID, _ := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	var created omsDomain.PackingSession
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: time.Now(),
		}
		return sessionRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var active *omsDomain.PackingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		active, err = sessionRepo.FindActiveBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, created.ID, active.ID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created.Status = omsDomain.PackingSessionStatusCompleted
		created.CompletedAt = &now
		created.CompletedBy = "user-1"
		return sessionRepo.Update(ctx, &created)
	})
	require.NoError(t, err)

	var found *omsDomain.PackingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = sessionRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.PackingSessionStatusCompleted, found.Status)
	require.Equal(t, "user-1", found.CompletedBy)

	// A completed session is no longer "active" for a new pack to resume.
	var activeAfterComplete *omsDomain.PackingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		activeAfterComplete, err = sessionRepo.FindActiveBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, activeAfterComplete)
}

func TestPackingSessionRepo_Integration_NotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPackingSessionRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := sessionRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, omsDomain.ErrPackingSessionNotFound)
}

func TestPackingSessionRepo_Integration_AcquireAndReleaseLock(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPackingSessionRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID, _ := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	var sessionID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		session := omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: time.Now(),
		}
		if err := sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		sessionID = session.ID
		return nil
	})
	require.NoError(t, err)

	lockedAt := time.Now().Truncate(time.Microsecond)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return sessionRepo.AcquireLock(ctx, sessionID, "user-1", lockedAt)
	})
	require.NoError(t, err)

	var found *omsDomain.PackingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = sessionRepo.FindByID(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.LockedBy)
	require.Equal(t, "user-1", *found.LockedBy)

	// ReleaseLock must actually clear locked_by/locked_at to NULL -- see
	// AcquireLock/ReleaseLock's doc comment for why a struct-based Update()
	// can't do this.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return sessionRepo.ReleaseLock(ctx, sessionID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = sessionRepo.FindByID(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found.LockedBy)
	require.Nil(t, found.LockedAt)
}

func TestPackScanRepo_Integration_CreateListBySessionOrdersByScannedAtAndDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPackingSessionRepo(db)
	scanRepo := repositories.NewPackScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID, lineItemID := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	var sessionID, firstScanID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		session := omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: time.Now(),
		}
		if err := sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		sessionID = session.ID

		older := time.Now().Add(-time.Hour)
		olderScan := omsDomain.PackScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: sessionID, LineItemID: &lineItemID, SKUID: skuID,
			Quantity: 1, ScannedAt: older, ScannedBy: "user-1",
		}
		if err := scanRepo.Create(ctx, &olderScan); err != nil {
			return err
		}
		firstScanID = olderScan.ID

		newer := time.Now()
		newerScan := omsDomain.PackScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: sessionID, LineItemID: &lineItemID, SKUID: skuID,
			Quantity: 1, ScannedAt: newer, ScannedBy: "user-1",
		}
		return scanRepo.Create(ctx, &newerScan)
	})
	require.NoError(t, err)

	var scans []omsDomain.PackScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListBySession(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 2)
	require.Equal(t, firstScanID, scans[0].ID, "oldest first")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scanRepo.Delete(ctx, firstScanID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListBySession(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 1, "a deleted scan no longer appears in the session's scan list")
}

// TestPackingSessionRepo_Integration_RLSIsolatesTenants proves the new
// tables' tenant_isolation policy actually enforces isolation, same
// reasoning as TestPickingSessionRepo_Integration_RLSIsolatesTenants.
func TestPackingSessionRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)
	orderID, _ := seedSalesOrderWithLineItemForPackingDB(t, superDB, ctxA, skuID, warehouseID)
	sessionRepo := repositories.NewPackingSessionRepo(db)

	var sessionID string
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		session := omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: time.Now(),
		}
		return sessionRepo.Create(ctx, &session)
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		s, err := sessionRepo.FindActiveBySalesOrder(ctx, orderID)
		if err != nil {
			return err
		}
		sessionID = s.ID
		return nil
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		_, err := sessionRepo.FindByID(ctx, sessionID)
		return err
	})
	require.ErrorIs(t, err, omsDomain.ErrPackingSessionNotFound, "tenant B must not see tenant A's packing session")
}

// Order changes recorded while packing round-trip every field, a freebie
// stores no line item, and MarkReplayed stamps a row once applied.
func TestPackScanRepo_Integration_OrderChangesRoundTripAndMarkReplayed(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPackingSessionRepo(db)
	scanRepo := repositories.NewPackScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID, lineItemID := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	covered, price := int64(1), int64(250)
	var substituteID, freebieID, plainID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		session := omsDomain.PackingSession{TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Status: omsDomain.PackingSessionStatusInProgress, StartedAt: time.Now()}
		if err := sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		base := time.Now().Add(-time.Minute)
		rows := []*omsDomain.PackScan{
			{TenantID: tenant.IDFromContext(ctx), SessionID: session.ID, LineItemID: &lineItemID, SKUID: skuID, Quantity: 1, ScannedAt: base},
			{TenantID: tenant.IDFromContext(ctx), SessionID: session.ID, LineItemID: &lineItemID, SKUID: skuID, ActionType: omsDomain.PackScanActionSubstitute,
				Quantity: 3, OriginalQtyCovered: &covered, ReturnToStock: true, Reason: "out of stock", ScannedAt: base.Add(time.Second)},
			{TenantID: tenant.IDFromContext(ctx), SessionID: session.ID, SKUID: skuID, ActionType: omsDomain.PackScanActionNewItem,
				Quantity: 1, PriceMinorUnits: &price, Reason: "Extra jar", ScannedAt: base.Add(2 * time.Second)},
		}
		for _, r := range rows {
			if err := scanRepo.Create(ctx, r); err != nil {
				return err
			}
		}
		plainID, substituteID, freebieID = rows[0].ID, rows[1].ID, rows[2].ID
		if rows[0].ActionType != omsDomain.PackScanActionPack {
			t.Errorf("an unset action is stored as pack, got %q", rows[0].ActionType)
		}
		return scanRepo.MarkReplayed(ctx, substituteID, time.Now())
	})
	require.NoError(t, err)

	var scans []omsDomain.PackScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sessions, err := sessionRepo.ListActiveBySalesOrders(ctx, []string{orderID})
		if err != nil {
			return err
		}
		scans, err = scanRepo.ListBySession(ctx, sessions[0].ID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 3)

	assert.Equal(t, plainID, scans[0].ID)
	assert.Equal(t, omsDomain.PackScanActionPack, scans[0].ActionType)

	sub := scans[1]
	assert.Equal(t, substituteID, sub.ID)
	assert.Equal(t, omsDomain.PackScanActionSubstitute, sub.ActionType)
	require.NotNil(t, sub.OriginalQtyCovered)
	assert.Equal(t, int64(1), *sub.OriginalQtyCovered)
	assert.True(t, sub.ReturnToStock)
	assert.Equal(t, "out of stock", sub.Reason)
	assert.NotNil(t, sub.ReplayedAt)

	extra := scans[2]
	assert.Equal(t, freebieID, extra.ID)
	assert.Nil(t, extra.LineItemID)
	require.NotNil(t, extra.PriceMinorUnits)
	assert.Equal(t, int64(250), *extra.PriceMinorUnits)
	assert.Nil(t, extra.ReplayedAt)
}

func TestPackScanRepo_Integration_SessionIDsWithScans(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPackingSessionRepo(db)
	scanRepo := repositories.NewPackScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID, lineItemID := seedSalesOrderWithLineItemForPackingDB(t, db, ctx, skuID, warehouseID)

	var withScan, empty string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, id := range []*string{&withScan, &empty} {
			s := omsDomain.PackingSession{TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Status: omsDomain.PackingSessionStatusCompleted, StartedAt: time.Now()}
			if err := sessionRepo.Create(ctx, &s); err != nil {
				return err
			}
			*id = s.ID
		}
		for i := 0; i < 2; i++ {
			if err := scanRepo.Create(ctx, &omsDomain.PackScan{TenantID: tenant.IDFromContext(ctx), SessionID: withScan, LineItemID: &lineItemID, SKUID: skuID, Quantity: 1, ScannedAt: time.Now()}); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var got []string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		got, err = scanRepo.SessionIDsWithScans(ctx, []string{withScan, empty})
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, []string{withScan}, got)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		got, err = scanRepo.SessionIDsWithScans(ctx, nil)
		return err
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}
