package pms_test

import (
	"context"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearCostPriceOverride_resetsSourceBackToComputed(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), skuRepo)

	sku, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.NotNil(t, sku)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceComputed, sku.CostPriceSource)
	assert.Equal(t, []string{"sku-1"}, skuRepo.clearOverrideCalls)
}

func TestClearCostPriceOverride_propagatesNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), skuRepo)

	_, err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
	assert.Empty(t, skuRepo.clearOverrideCalls)
}
