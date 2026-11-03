package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubLastReceiptCost struct {
	last *apppms.LastReceiptCost
	err  error
}

func (s stubLastReceiptCost) LastReceiptCost(context.Context, string) (*apppms.LastReceiptCost, error) {
	return s.last, s.err
}

func lastReceipt(cost int64) *apppms.LastReceiptCost {
	return &apppms.LastReceiptCost{LandedUnitCostMinorUnits: cost, PurchaseOrderLineItemID: "line-9"}
}

func TestSyncCostPriceFromHistory_followsTheLastReceiptStillStanding(t *testing.T) {
	cost := int64(800)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed}}
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewSyncCostPriceFromHistoryUseCase(newMockGormDB(t), skuRepo, stubLastReceiptCost{last: lastReceipt(500)}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "sku-1"))

	require.Len(t, skuRepo.updateCalls, 1)
	assert.Equal(t, int64(500), skuRepo.updateCalls[0].costPriceMinorUnits)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.sku.cost_price_changed", e.Subject)
	assert.Equal(t, map[string]any{"reason": "receipt", "reference_type": "purchase_order_line_item", "reference_id": "line-9"}, e.Data())
}

func TestSyncCostPriceFromHistory_changesNothing(t *testing.T) {
	cost := int64(500)
	for name, tc := range map[string]struct {
		sku  pmsDomain.SKU
		last *apppms.LastReceiptCost
	}{
		"already the last receipt": {pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost}, lastReceipt(500)},
		"manual cost wins":         {pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost, CostPriceSource: pmsDomain.SKUCostPriceSourceManual}, lastReceipt(700)},
		"no paid receipt":          {pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			sku := tc.sku
			skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &sku}
			outbox := &eventrecordtest.Outbox{}
			uc := apppms.NewSyncCostPriceFromHistoryUseCase(newMockGormDB(t), skuRepo, stubLastReceiptCost{last: tc.last}, outbox)

			require.NoError(t, uc.Execute(context.Background(), "sku-1"))

			assert.Empty(t, skuRepo.updateCalls)
			assert.Empty(t, outbox.Events)
		})
	}
}

func TestSyncCostPriceFromHistory_propagatesTheLookupError(t *testing.T) {
	boom := errors.New("boom")
	uc := apppms.NewSyncCostPriceFromHistoryUseCase(newMockGormDB(t), &stubSKURepoForReceiptCost{}, stubLastReceiptCost{err: boom}, &eventrecordtest.Outbox{})

	require.ErrorIs(t, uc.Execute(context.Background(), "sku-1"), boom)
}
