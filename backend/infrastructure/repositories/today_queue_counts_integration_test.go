//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSalesOrderRepo_Integration_Count_matchesListTotalPerQueue proves Count
// applies List's exact filter rules for every dashboard work queue -- and
// that it stays tenant-scoped (another tenant's orders never count).
func TestSalesOrderRepo_Integration_Count_matchesListTotalPerQueue(t *testing.T) {
	db, superDSN := newIntegrationDBAndDSN(t)
	// Reads go through the non-superuser app role -- the superuser
	// connection ignores RLS, so it couldn't prove tenant isolation.
	appDB := newAppRoleDB(t, superDSN)
	appRepo := repositories.NewSalesOrderRepo(appDB)
	_, ctx := newTestTenant()
	_, otherCtx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	_, otherWarehouseID := seedProductSKUWarehouse(t, db, otherCtx)
	repo := repositories.NewSalesOrderRepo(db)

	type spec struct {
		extID                     string
		source                    omsDomain.SalesOrderSource
		status                    omsDomain.SalesOrderStatus
		fulfilled, packed         bool
		needsStock, needsSKUCheck bool
	}
	seed := func(ctx context.Context, warehouseID string, specs []spec) {
		t.Helper()
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			for _, s := range specs {
				o := omsDomain.SalesOrder{
					TenantID: tenant.IDFromContext(ctx), Source: s.source, ExternalOrderID: s.extID, WarehouseID: warehouseID,
					CurrencyCode: "USD", Status: s.status, Fulfilled: s.fulfilled, Packed: s.packed,
					NeedsStockReview: s.needsStock, NeedsSKUReview: s.needsSKUCheck, OccurredAt: now, CreatedAt: now,
				}
				if err := repo.Create(ctx, &o); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err)
	}
	woo, pos := omsDomain.SalesOrderSourceWooCommerce, omsDomain.SalesOrderSourcePOS
	processing, onHold := omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold
	seed(ctx, warehouseID, []spec{
		{extID: "tq-new", source: woo, status: processing},                                                // to pick + to pack
		{extID: "tq-onhold", source: woo, status: onHold, needsStock: true},                               // to pick + to pack + stock issue
		{extID: "tq-picked", source: woo, status: processing, fulfilled: true},                            // to pack only
		{extID: "tq-packed", source: woo, status: processing, fulfilled: true, packed: true},              // neither
		{extID: "tq-pending", source: woo, status: omsDomain.SalesOrderStatusPending},                     // unpaid -- neither
		{extID: "tq-done", source: woo, status: omsDomain.SalesOrderStatusCompleted, needsSKUCheck: true}, // unmatched only
		{extID: "tq-pos", source: pos, status: processing, needsStock: true},                              // POS: stock issue only
	})
	// Another tenant's queue-matching order must never be counted.
	seed(otherCtx, otherWarehouseID, []spec{
		{extID: "tq-other", source: woo, status: processing, needsStock: true, needsSKUCheck: true},
	})

	flagged := true
	cases := []struct {
		name   string
		filter omsDomain.ListSalesOrdersFilter
		want   int64
	}{
		{"awaiting picking", omsDomain.ListSalesOrdersFilter{AwaitingPicking: true}, 2},
		{"awaiting packing", omsDomain.ListSalesOrdersFilter{AwaitingPacking: true}, 3},
		{"needs stock review", omsDomain.ListSalesOrdersFilter{NeedsStockReview: &flagged}, 2},
		{"needs sku review", omsDomain.ListSalesOrdersFilter{NeedsSKUReview: &flagged}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var count, listTotal int64
			err := tenant.WithTenant(ctx, appDB, func(ctx context.Context) error {
				var err error
				if count, err = appRepo.Count(ctx, c.filter); err != nil {
					return err
				}
				_, listTotal, err = appRepo.List(ctx, c.filter)
				return err
			})
			require.NoError(t, err)
			assert.Equal(t, c.want, count)
			assert.Equal(t, listTotal, count, "Count must match List's total for the same filter")
		})
	}
}

// TestBatchRepo_Integration_QueueCounts_matchListLengths proves
// CountExpiringSoon/CountPendingReview are len(ListExpiringSoon)/
// len(ListPendingReview): in-stock-only + window for expiring, expired and
// unreviewed for pending.
func TestBatchRepo_Integration_QueueCounts_matchListLengths(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, whID := seedProductSKUWarehouse(t, db, ctx)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)

	now := time.Now()
	soon := now.Add(3 * 24 * time.Hour)
	far := now.Add(60 * 24 * time.Hour)
	past := now.Add(-10 * 24 * time.Hour)

	inStockSoon := makeBatch(t, db, ctx, skuID, whID, &soon)
	makeBatch(t, db, ctx, skuID, whID, &soon) // no stock -- not expiring-soon
	inStockFar := makeBatch(t, db, ctx, skuID, whID, &far)
	expiredPending := makeBatch(t, db, ctx, skuID, whID, &past)
	expiredReviewed := makeBatch(t, db, ctx, skuID, whID, &past)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, id := range []string{inStockSoon, inStockFar, expiredPending} {
			batchID := id
			m := invDomain.StockMovement{
				TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: whID, BatchID: &batchID,
				Type: invDomain.MovementTypeReceipt, QuantityDelta: 4, OccurredAt: now, CreatedAt: now,
			}
			if err := movementRepo.Create(ctx, &m); err != nil {
				return err
			}
		}
		for _, id := range []string{expiredPending, expiredReviewed} {
			if err := batchRepo.MarkExpired(ctx, id); err != nil {
				return err
			}
		}
		return batchRepo.MarkReviewed(ctx, expiredReviewed)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		expiring, err := batchRepo.CountExpiringSoon(ctx, 7)
		require.NoError(t, err)
		expiringList, err := batchRepo.ListExpiringSoon(ctx, 7)
		require.NoError(t, err)
		// inStockSoon + expiredPending (past expiry, still in stock -- the
		// list has no lower bound).
		assert.Equal(t, int64(2), expiring)
		assert.Equal(t, int64(len(expiringList)), expiring)

		pending, err := batchRepo.CountPendingReview(ctx)
		require.NoError(t, err)
		pendingList, err := batchRepo.ListPendingReview(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), pending)
		assert.Equal(t, int64(len(pendingList)), pending)
		return nil
	})
	require.NoError(t, err)
}
