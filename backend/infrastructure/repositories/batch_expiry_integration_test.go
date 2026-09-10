//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func makeBatch(t *testing.T, db *gorm.DB, ctx context.Context, skuID, warehouseID string, expiryDate *time.Time) string {
	t.Helper()
	batchRepo := repositories.NewBatchRepo(db)
	var batchID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		b := &invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-1", ExpiryDate: expiryDate, ReceivedAt: now,
			Status: invDomain.BatchStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, b); err != nil {
			return err
		}
		batchID = b.ID
		return nil
	})
	require.NoError(t, err)
	return batchID
}

func setBatchExpiryThreshold(t *testing.T, db *gorm.DB, ctx context.Context, days int) {
	t.Helper()
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return settingsRepo.Upsert(ctx, &settingsDomain.TenantSettings{
			TenantID: tenant.IDFromContext(ctx), BatchExpiryThresholdDays: days, UpdatedAt: time.Now(),
		})
	})
	require.NoError(t, err)
}

func TestBatchRepo_Integration_ListToExpire_appliesPerTenantThresholdAndSkipsAlreadyExpired(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	batchRepo := repositories.NewBatchRepo(db)

	skuA, whA := seedProductSKUWarehouse(t, db, ctxA)
	skuB, whB := seedProductSKUWarehouse(t, db, ctxB)

	setBatchExpiryThreshold(t, db, ctxA, 7)
	setBatchExpiryThreshold(t, db, ctxB, 0) // disabled -- must never appear

	pastThreshold := time.Now().Add(-10 * 24 * time.Hour)
	withinThreshold := time.Now().Add(-3 * 24 * time.Hour)

	dueID := makeBatch(t, db, ctxA, skuA, whA, &pastThreshold)
	makeBatch(t, db, ctxA, skuA, whA, &withinThreshold) // not past threshold yet -- excluded
	makeBatch(t, db, ctxA, skuA, whA, nil)              // no expiry -- excluded
	alreadyExpiredID := makeBatch(t, db, ctxA, skuA, whA, &pastThreshold)
	makeBatch(t, db, ctxB, skuB, whB, &pastThreshold) // tenant B threshold disabled -- excluded

	// Flip alreadyExpiredID to Expired before listing -- it must not
	// reappear even though it otherwise matches the threshold criteria.
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return batchRepo.MarkExpired(ctx, alreadyExpiredID)
	})
	require.NoError(t, err)

	var found []invDomain.BatchToExpire
	err = tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.ListToExpire(ctx)
		return err
	})
	require.NoError(t, err)

	foundIDs := make(map[string]bool, len(found))
	for _, b := range found {
		foundIDs[b.BatchID] = true
	}
	require.True(t, foundIDs[dueID], "batch past its tenant's threshold must be found")
	require.Len(t, found, 1, "within-threshold, no-expiry, already-expired, and disabled-tenant batches must all be excluded")
	require.Equal(t, skuA, found[0].SKUID, "SKUID must be populated -- ExpireBatchUseCase needs it to enqueue inventory.stock.changed")
}

func TestBatchRepo_Integration_ListPendingReview_returnsOnlyExpiredUnreviewedBatches(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	batchRepo := repositories.NewBatchRepo(db)
	skuID, whID := seedProductSKUWarehouse(t, db, ctx)

	expiry := time.Now().Add(-10 * 24 * time.Hour)
	pendingID := makeBatch(t, db, ctx, skuID, whID, &expiry)
	reviewedID := makeBatch(t, db, ctx, skuID, whID, &expiry)
	makeBatch(t, db, ctx, skuID, whID, &expiry) // still Active -- never flagged expired, must be excluded

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := batchRepo.MarkExpired(ctx, pendingID); err != nil {
			return err
		}
		if err := batchRepo.MarkExpired(ctx, reviewedID); err != nil {
			return err
		}
		return batchRepo.MarkReviewed(ctx, reviewedID)
	})
	require.NoError(t, err)

	var found []invDomain.ExpiredBatchReview
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.ListPendingReview(ctx)
		return err
	})
	require.NoError(t, err)

	foundIDs := make(map[string]bool, len(found))
	for _, b := range found {
		foundIDs[b.BatchID] = true
	}
	require.True(t, foundIDs[pendingID], "expired, not-yet-reviewed batch must be found")
	require.Len(t, found, 1, "already-reviewed and never-expired batches must be excluded")
}

func TestBatchRepo_Integration_MarkReviewed_removesFromPendingReview(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	batchRepo := repositories.NewBatchRepo(db)
	skuID, whID := seedProductSKUWarehouse(t, db, ctx)

	expiry := time.Now().Add(-10 * 24 * time.Hour)
	batchID := makeBatch(t, db, ctx, skuID, whID, &expiry)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := batchRepo.MarkExpired(ctx, batchID); err != nil {
			return err
		}
		return batchRepo.MarkReviewed(ctx, batchID)
	})
	require.NoError(t, err)

	var found []invDomain.ExpiredBatchReview
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.ListPendingReview(ctx)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, found)
}

func TestBatchRepo_Integration_ExtendExpiry_setsNewDateAndReactivates(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	batchRepo := repositories.NewBatchRepo(db)
	skuID, whID := seedProductSKUWarehouse(t, db, ctx)

	expiry := time.Now().Add(-10 * 24 * time.Hour)
	batchID := makeBatch(t, db, ctx, skuID, whID, &expiry)
	// expiry_date is a DATE column -- pgx's DATE encoder takes Year/Month/Day
	// straight from time.Time's own Location without normalizing to UTC
	// first (same gotcha documented on pms_sku_sale_integration_test.go),
	// so build a UTC-midnight value the same way production does via
	// openapi_types.Date -- a local-tz time.Now()-derived value would
	// round-trip to the wrong calendar day near local midnight.
	newExpiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := batchRepo.MarkExpired(ctx, batchID); err != nil {
			return err
		}
		return batchRepo.ExtendExpiry(ctx, batchID, newExpiry)
	})
	require.NoError(t, err)

	var found *invDomain.Batch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.FindByID(ctx, batchID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, invDomain.BatchStatusActive, found.Status)
	require.WithinDuration(t, newExpiry, *found.ExpiryDate, time.Second)

	var pending []invDomain.ExpiredBatchReview
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		pending, err = batchRepo.ListPendingReview(ctx)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, pending, "re-activated batch must drop out of the review queue")
}

func TestBatchRepo_Integration_MarkExpired_persistsStatus(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	batchRepo := repositories.NewBatchRepo(db)
	skuID, whID := seedProductSKUWarehouse(t, db, ctx)

	expiry := time.Now().Add(-30 * 24 * time.Hour)
	batchID := makeBatch(t, db, ctx, skuID, whID, &expiry)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return batchRepo.MarkExpired(ctx, batchID)
	})
	require.NoError(t, err)

	var found *invDomain.Batch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = batchRepo.FindByID(ctx, batchID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, invDomain.BatchStatusExpired, found.Status)
}
