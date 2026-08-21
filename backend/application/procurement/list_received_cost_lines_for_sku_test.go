package procurement_test

import (
	"context"
	"testing"

	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReceivedCostLinesForSKU_returnsLandedCostPerMatchingLineAcrossReceivedPOs(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	lineItemRepo := &stubLineItemRepo{
		poIDsBySKU: map[string][]string{skuID: {"po-1", "po-2"}},
		byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
			"po-1": {
				{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &skuID, Quantity: 10, UnitCostMinorUnits: 800, LineType: procurementDomain.LineItemTypeProduct},
				{ID: "line-shipping", PurchaseOrderID: "po-1", Quantity: 1, UnitCostMinorUnits: 1000, LineType: procurementDomain.LineItemTypeShipping},
			},
			"po-2": {
				{ID: "line-2", PurchaseOrderID: "po-2", SKUID: &skuID, Quantity: 5, UnitCostMinorUnits: 500, LineType: procurementDomain.LineItemTypeProduct},
			},
		},
	}
	uc := appprocurement.NewListReceivedCostLinesForSKUUseCase(db, lineItemRepo)

	lines, err := uc.Execute(context.Background(), skuID)

	require.NoError(t, err)
	require.Len(t, lines, 2)
	assert.Contains(t, lines, pmsapp.PurchaseCostLine{PurchaseOrderLineItemID: "line-1", Quantity: 10, LandedUnitCostMinorUnits: 900})
	assert.Contains(t, lines, pmsapp.PurchaseCostLine{PurchaseOrderLineItemID: "line-2", Quantity: 5, LandedUnitCostMinorUnits: 500})
}

func TestListReceivedCostLinesForSKU_skipsLineItemsForOtherSKUsAndNonProductRows(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	otherSKU := "sku-2"
	lineItemRepo := &stubLineItemRepo{
		poIDsBySKU: map[string][]string{skuID: {"po-1"}},
		byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
			"po-1": {
				{ID: "line-1", PurchaseOrderID: "po-1", SKUID: &skuID, Quantity: 10, UnitCostMinorUnits: 800, LineType: procurementDomain.LineItemTypeProduct},
				{ID: "line-other-sku", PurchaseOrderID: "po-1", SKUID: &otherSKU, Quantity: 4, UnitCostMinorUnits: 300, LineType: procurementDomain.LineItemTypeProduct},
				{ID: "line-fee", PurchaseOrderID: "po-1", SKUID: &skuID, Quantity: 1, UnitCostMinorUnits: 50, LineType: procurementDomain.LineItemTypeFee},
			},
		},
	}
	uc := appprocurement.NewListReceivedCostLinesForSKUUseCase(db, lineItemRepo)

	lines, err := uc.Execute(context.Background(), skuID)

	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, "line-1", lines[0].PurchaseOrderLineItemID)
}

func TestListReceivedCostLinesForSKU_emptyWhenNoReceivedPurchaseOrders(t *testing.T) {
	db := newMockGormDB(t)
	lineItemRepo := &stubLineItemRepo{}
	uc := appprocurement.NewListReceivedCostLinesForSKUUseCase(db, lineItemRepo)

	lines, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Empty(t, lines)
}
