//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSKURepo_Integration_InventoryCostIsSeparateFromCostPrice: the moving
// average lives in its own column, and writing it leaves a manual cost
// price (and its source) alone.
func TestSKURepo_Integration_InventoryCostIsSeparateFromCostPrice(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	skuRepo := repositories.NewSKURepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := skuRepo.SetManualCostPrice(ctx, skuID, 900); err != nil {
			return err
		}
		return skuRepo.UpdateInventoryCost(ctx, skuID, 467)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sku, err := skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		require.NotNil(t, sku.InventoryCostMinorUnits)
		assert.Equal(t, int64(467), *sku.InventoryCostMinorUnits)
		require.NotNil(t, sku.CostPriceMinorUnits)
		assert.Equal(t, int64(900), *sku.CostPriceMinorUnits)
		assert.Equal(t, pmsDomain.SKUCostPriceSourceManual, sku.CostPriceSource)
		return nil
	})
	require.NoError(t, err)
}
