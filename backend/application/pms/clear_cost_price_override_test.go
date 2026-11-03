package pms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearCostPriceOverride_resetsSourceBackToComputed(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), skuRepo, stubLastReceiptCost{}, &eventrecordtest.Outbox{})

	sku, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.NotNil(t, sku)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceComputed, sku.CostPriceSource)
	assert.Equal(t, []string{"sku-1"}, skuRepo.clearOverrideCalls)
}

func TestClearCostPriceOverride_propagatesNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), skuRepo, stubLastReceiptCost{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
	assert.Empty(t, skuRepo.clearOverrideCalls)
}

// Cleared, the cost price goes straight back to the last receipt still
// standing in the supplier cost history (COM-11).
func TestClearCostPriceOverride_goesBackToTheLastReceipt(t *testing.T) {
	manual := int64(999)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &manual, CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), skuRepo, stubLastReceiptCost{last: lastReceipt(520)}, outbox)

	sku, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"sku-1"}, skuRepo.clearOverrideCalls)
	require.Len(t, skuRepo.updateCostPriceCalls, 1)
	assert.Equal(t, int64(520), skuRepo.updateCostPriceCalls[0].CostPriceMinorUnits)
	require.NotNil(t, sku.CostPriceMinorUnits)
	assert.Equal(t, int64(520), *sku.CostPriceMinorUnits)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceComputed, sku.CostPriceSource)
	e := onlyEvent(t, outbox)
	assert.Equal(t, map[string][2]any{
		"cost_price_minor_units": {float64(999), float64(520)},
		"cost_price_source":      {"manual", "computed"},
	}, e.Changes())
	assert.Equal(t, "override_cleared", e.Data()["reason"])
}

func TestClearCostPriceOverride_propagatesTheLookupError(t *testing.T) {
	boom := errors.New("boom")
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), skuRepo, stubLastReceiptCost{err: boom}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "sku-1")

	require.ErrorIs(t, err, boom)
	assert.Empty(t, skuRepo.clearOverrideCalls)
}
