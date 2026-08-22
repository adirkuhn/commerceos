//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func seedSalesOrderForPickingDB(t *testing.T, db *gorm.DB, ctx context.Context, warehouseID string) string {
	t.Helper()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	var orderID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-picking-" + tenant.IDFromContext(ctx), WarehouseID: warehouseID,
			CurrencyCode: "USD", Status: omsDomain.SalesOrderStatusProcessing,
			TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		orderID = order.ID
		return nil
	})
	require.NoError(t, err)
	return orderID
}

func TestPickingSessionRepo_Integration_CreateFindActiveUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPickingSessionRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	var created omsDomain.PickingSession
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now(),
		}
		return sessionRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var active *omsDomain.PickingSession
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
		created.Status = omsDomain.PickingSessionStatusCompleted
		created.CompletedAt = &now
		created.CompletedBy = "user-1"
		return sessionRepo.Update(ctx, &created)
	})
	require.NoError(t, err)

	var found *omsDomain.PickingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = sessionRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, omsDomain.PickingSessionStatusCompleted, found.Status)
	require.Equal(t, "user-1", found.CompletedBy)

	// Completed sessions are no longer "active" for a new pick to resume.
	var activeAfterComplete *omsDomain.PickingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		activeAfterComplete, err = sessionRepo.FindActiveBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, activeAfterComplete)
}

func TestPickingSessionRepo_Integration_FindActiveMatchesPausedTooButNotCompleted(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPickingSessionRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		paused := time.Now()
		session := omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PickingSessionStatusPaused, StartedAt: time.Now().Add(-time.Hour), PausedAt: &paused,
		}
		return sessionRepo.Create(ctx, &session)
	})
	require.NoError(t, err)

	var active *omsDomain.PickingSession
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		active, err = sessionRepo.FindActiveBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, active, "a paused session is still active for resume purposes")
	require.Equal(t, omsDomain.PickingSessionStatusPaused, active.Status)
}

func TestPickingSessionRepo_Integration_NotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPickingSessionRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := sessionRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}

func TestPickScanRepo_Integration_CreateListBySessionOrdersByScannedAt(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPickingSessionRepo(db)
	scanRepo := repositories.NewPickScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	var sessionID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		session := omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now(),
		}
		if err := sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		sessionID = session.ID

		older := time.Now().Add(-time.Hour)
		olderScan := omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: sessionID, SKUID: skuID,
			ActionType: omsDomain.PickScanActionPick, Quantity: 1, ScannedAt: older, ScannedBy: "user-1",
		}
		if err := scanRepo.Create(ctx, &olderScan); err != nil {
			return err
		}

		newer := time.Now()
		price := int64(500)
		newerScan := omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: sessionID, SKUID: skuID,
			ActionType: omsDomain.PickScanActionFreebie, Quantity: 2, PriceMinorUnits: &price,
			ScannedAt: newer, ScannedBy: "user-1",
		}
		return scanRepo.Create(ctx, &newerScan)
	})
	require.NoError(t, err)

	var scans []omsDomain.PickScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListBySession(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 2)
	require.Equal(t, omsDomain.PickScanActionPick, scans[0].ActionType, "oldest first")
	require.Equal(t, omsDomain.PickScanActionFreebie, scans[1].ActionType)
	require.NotNil(t, scans[1].PriceMinorUnits)
	require.Equal(t, int64(500), *scans[1].PriceMinorUnits)
	require.Nil(t, scans[0].ReplayedAt, "a freshly-staged scan is not replayed yet")
}

func TestPickScanRepo_Integration_MarkReplayed(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPickingSessionRepo(db)
	scanRepo := repositories.NewPickScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	var sessionID, scanID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		session := omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now(),
		}
		if err := sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		sessionID = session.ID

		scan := omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: sessionID, SKUID: skuID,
			ActionType: omsDomain.PickScanActionPick, Quantity: 1, ScannedAt: time.Now(), ScannedBy: "user-1",
		}
		if err := scanRepo.Create(ctx, &scan); err != nil {
			return err
		}
		scanID = scan.ID
		return nil
	})
	require.NoError(t, err)

	replayedAt := time.Now().Truncate(time.Microsecond)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scanRepo.MarkReplayed(ctx, scanID, replayedAt)
	})
	require.NoError(t, err)

	var scans []omsDomain.PickScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListBySession(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 1)
	require.NotNil(t, scans[0].ReplayedAt, "MarkReplayed must persist -- a retried Finish needs this to skip the row")
	require.WithinDuration(t, replayedAt, *scans[0].ReplayedAt, time.Second)
}

func TestPickScanRepo_Integration_Delete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	sessionRepo := repositories.NewPickingSessionRepo(db)
	scanRepo := repositories.NewPickScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	var sessionID, scanID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		session := omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now(),
		}
		if err := sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		sessionID = session.ID

		scan := omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: sessionID, SKUID: skuID,
			ActionType: omsDomain.PickScanActionPick, Quantity: 1, ScannedAt: time.Now(), ScannedBy: "user-1",
		}
		if err := scanRepo.Create(ctx, &scan); err != nil {
			return err
		}
		scanID = scan.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scanRepo.Delete(ctx, scanID)
	})
	require.NoError(t, err)

	var scans []omsDomain.PickScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListBySession(ctx, sessionID)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, scans, "a deleted staged scan no longer appears in the session's scan list")
}

// TestPickingSessionRepo_Integration_RLSIsolatesTenants proves the new
// tables' tenant_isolation policy actually enforces isolation (not just
// that the SQL runs) -- routed through commerceos_app, not the
// migration-owning superuser, same reasoning as
// TestSKUCostPriceHistoryRepo_Integration_RLSIsolatesTenants.
func TestPickingSessionRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)
	orderID := seedSalesOrderForPickingDB(t, superDB, ctxA, warehouseID)
	sessionRepo := repositories.NewPickingSessionRepo(db)

	var sessionID string
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		session := omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now(),
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
	require.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound, "tenant B must not see tenant A's picking session")
}
