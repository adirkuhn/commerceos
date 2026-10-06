package pms_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetManualCostPrice_writesCostPriceAndFlipsSourceToManual(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	uc := apppms.NewSetManualCostPriceUseCase(newMockGormDB(t), skuRepo, &eventrecordtest.Outbox{})

	sku, err := uc.Execute(context.Background(), "sku-1", 1250)

	require.NoError(t, err)
	require.NotNil(t, sku)
	require.NotNil(t, sku.CostPriceMinorUnits)
	assert.Equal(t, int64(1250), *sku.CostPriceMinorUnits)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceManual, sku.CostPriceSource)
	require.Len(t, skuRepo.setManualCostPriceCalls, 1)
	assert.Equal(t, "sku-1", skuRepo.setManualCostPriceCalls[0].SKUID)
	assert.Equal(t, int64(1250), skuRepo.setManualCostPriceCalls[0].CostPriceMinorUnits)
}

func TestSetManualCostPrice_rejectsNegativeCostPrice(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	uc := apppms.NewSetManualCostPriceUseCase(newMockGormDB(t), skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "sku-1", -1)

	require.ErrorIs(t, err, pmsDomain.ErrInvalidCostPrice)
	assert.Empty(t, skuRepo.setManualCostPriceCalls)
}

func TestSetManualCostPrice_propagatesNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewSetManualCostPriceUseCase(newMockGormDB(t), skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing", 1000)

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
	assert.Empty(t, skuRepo.setManualCostPriceCalls)
}
