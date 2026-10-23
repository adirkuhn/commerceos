package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// Unlike a manual override, this cost stays automatic: the next goods
// receipt blends into it and moves it.
func TestSetAutomaticCostPrice_writesCostAndKeepsItAutomatic(t *testing.T) {
	old := int64(300)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{
		ID: "sku-1", CostPriceMinorUnits: &old, CostPriceSource: pmsDomain.SKUCostPriceSourceManual,
	}}
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewSetAutomaticCostPriceUseCase(newMockGormDB(t), skuRepo, outbox)

	sku, err := uc.Execute(context.Background(), "sku-1", 450)

	require.NoError(t, err)
	assert.Equal(t, int64(450), *sku.CostPriceMinorUnits)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceComputed, sku.CostPriceSource)
	require.Len(t, skuRepo.updateCalls, 1)
	assert.Equal(t, int64(450), skuRepo.updateCalls[0].costPriceMinorUnits)

	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "pms.sku.cost_price_changed", event.Subject)
	assert.Equal(t, "set", event.Data()["reason"])
	assert.Equal(t, [2]any{float64(300), float64(450)}, event.Changes()["cost_price_minor_units"])
	assert.Equal(t, [2]any{"manual", "computed"}, event.Changes()["cost_price_source"])
}

func TestSetAutomaticCostPrice_rejectsNegativeCost(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	uc := apppms.NewSetAutomaticCostPriceUseCase(newMockGormDB(t), skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "sku-1", -1)

	require.ErrorIs(t, err, pmsDomain.ErrInvalidCostPrice)
	assert.Empty(t, skuRepo.updateCalls)
}

func TestSetAutomaticCostPrice_propagatesNotFound(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := apppms.NewSetAutomaticCostPriceUseCase(newMockGormDB(t), skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing", 100)

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}
