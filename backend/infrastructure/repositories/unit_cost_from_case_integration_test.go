//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSyncUnitCostFromCase_Integration_UnitTakesCaseCost runs the real use
// case against real Postgres: a unit with no cost, sold out of a 20-unit
// case costing 13.99, gets 0.70 -- and the change is kept as a case_link
// cost ledger row (the ledger's source_type CHECK must allow it).
func TestSyncUnitCostFromCase_Integration_UnitTakesCaseCost(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	unitSKUID, _ := seedProductSKUWarehouse(t, db, ctx)

	skuRepo := repositories.NewSKURepo(db)
	ledgerRepo := repositories.NewSKUCostPriceLedgerRepo(db)
	batchLister := appinventory.NewListBatchesUseCase(db, repositories.NewBatchRepo(db),
		repositories.NewStockMovementRepo(db), repositories.NewWarehouseRepo(db))

	var caseSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		unitSKU, err := skuRepo.FindByID(ctx, unitSKUID)
		if err != nil {
			return err
		}
		caseSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: unitSKU.VariantID, SKUCode: "ITEST-CASE-20",
			CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive,
			CaseOfSKUID: &unitSKUID, UnitsPerCase: int64Ptr(20), CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, caseSKU); err != nil {
			return err
		}
		caseSKUID = caseSKU.ID
		return skuRepo.UpdateCostPriceFromReceipt(ctx, caseSKUID, 1399)
	})
	require.NoError(t, err)

	uc := apppms.NewSyncUnitCostFromCaseUseCase(db, skuRepo, repositories.NewVariantRepo(db), ledgerRepo,
		batchLister, repositories.NewOutboxRepository(db))
	require.NoError(t, uc.SyncFromCase(ctx, caseSKUID))
	require.NoError(t, uc.SyncFromCase(ctx, caseSKUID), "a second run changes nothing")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		unit, err := skuRepo.FindByID(ctx, unitSKUID)
		if err != nil {
			return err
		}
		require.NotNil(t, unit.CostPriceMinorUnits)
		assert.Equal(t, int64(70), *unit.CostPriceMinorUnits)
		assert.Equal(t, pmsDomain.SKUCostPriceSourceComputed, unit.CostPriceSource)

		entries, err := ledgerRepo.ListBySKU(ctx, unitSKUID, 0)
		if err != nil {
			return err
		}
		require.Len(t, entries, 1)
		assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceCaseLink, entries[0].SourceType)
		require.NotNil(t, entries[0].ReferenceID)
		assert.Equal(t, caseSKUID, *entries[0].ReferenceID)
		return nil
	})
	require.NoError(t, err)
}
