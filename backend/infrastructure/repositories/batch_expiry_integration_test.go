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
