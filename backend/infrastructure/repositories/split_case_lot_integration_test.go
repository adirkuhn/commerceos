//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSplitCaseIntoUnits_Integration_UnitsInheritTheCaseLot runs the real
// use case against real Postgres: splitting cases out of a lot must land
// the units in a unit-SKU batch with the same lot number and expiry, and a
// second split of the same lot must top up that batch, not create another.
func TestSplitCaseIntoUnits_Integration_UnitsInheritTheCaseLot(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	unitSKUID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	skuRepo := repositories.NewSKURepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	expiry := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	var caseSKUID, caseBatchID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		unitSKU, err := skuRepo.FindByID(ctx, unitSKUID)
		if err != nil {
			return err
		}
		caseSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: unitSKU.VariantID, SKUCode: "ITEST-CASE-36",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
			CaseOfSKUID: &unitSKUID, UnitsPerCase: int64Ptr(36), CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, caseSKU); err != nil {
			return err
		}
		caseSKUID = caseSKU.ID

		caseBatch := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: caseSKUID, WarehouseID: warehouseID,
			LotNumber: "300326", ExpiryDate: &expiry, ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &caseBatch); err != nil {
			return err
		}
		caseBatchID = caseBatch.ID
		return movementRepo.Create(ctx, &invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: caseSKUID, WarehouseID: warehouseID, BatchID: &caseBatchID,
			Type: invDomain.MovementTypeReceipt, QuantityDelta: 3, OccurredAt: now, CreatedAt: now,
		})
	})
	require.NoError(t, err)

	uc := appinventory.NewSplitCaseIntoUnitsUseCase(db, skuRepo, batchRepo, movementRepo, repositories.NewOutboxRepository(db), nil)
	split := func() *appinventory.SplitCaseIntoUnitsResult {
		result, err := uc.Execute(ctx, appinventory.SplitCaseIntoUnitsInput{
			CaseSKUID: caseSKUID, WarehouseID: warehouseID, BatchID: &caseBatchID, CasesToSplit: 1, OccurredAt: time.Now(),
		})
		require.NoError(t, err)
		return result
	}
	first := split()
	second := split()

	require.NotNil(t, first.UnitInMovement.BatchID)
	require.NotNil(t, second.UnitInMovement.BatchID)
	assert.Equal(t, *first.UnitInMovement.BatchID, *second.UnitInMovement.BatchID, "second split must reuse the unit batch")

	var unitBatches []invDomain.Batch
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		unitBatches, err = batchRepo.ListBySKU(ctx, unitSKUID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, unitBatches, 1)
	assert.Equal(t, "300326", unitBatches[0].LotNumber)
	require.NotNil(t, unitBatches[0].ExpiryDate)
	assert.True(t, expiry.Equal(*unitBatches[0].ExpiryDate))
	assert.Equal(t, warehouseID, unitBatches[0].WarehouseID)
}
