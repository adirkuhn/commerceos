package procurement_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestAllocateLandedUnitCost_splitsShippingProportionalToLineValue(t *testing.T) {
	// Two product lines: $80 worth (10 x $8) and $20 worth (5 x $4), plus a
	// $10 shipping line. Shipping should split 80/20 -> $8 to line A, $2 to
	// line B.
	lineItems := []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-a", LineType: procurementDomain.LineItemTypeProduct, Quantity: 10, UnitCostMinorUnits: 800},
		{ID: "line-b", LineType: procurementDomain.LineItemTypeProduct, Quantity: 5, UnitCostMinorUnits: 400},
		{ID: "line-shipping", LineType: procurementDomain.LineItemTypeShipping, Quantity: 1, UnitCostMinorUnits: 1000},
	}

	costA, err := appprocurement.AllocateLandedUnitCost(lineItems, "line-a")
	require.NoError(t, err)
	require.Equal(t, int64(880), costA) // 800 + (800/1000 * 1000)/10 = 800 + 80

	costB, err := appprocurement.AllocateLandedUnitCost(lineItems, "line-b")
	require.NoError(t, err)
	require.Equal(t, int64(440), costB) // 400 + (200/1000 * 1000)/5 = 400 + 40
}

func TestAllocateLandedUnitCost_includesFeeAndOtherChargeButNotDiscount(t *testing.T) {
	lineItems := []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-a", LineType: procurementDomain.LineItemTypeProduct, Quantity: 1, UnitCostMinorUnits: 1000},
		{ID: "line-fee", LineType: procurementDomain.LineItemTypeFee, Quantity: 1, UnitCostMinorUnits: 100},
		{ID: "line-other", LineType: procurementDomain.LineItemTypeOtherCharge, Quantity: 1, UnitCostMinorUnits: 50},
		{ID: "line-discount", LineType: procurementDomain.LineItemTypeDiscount, Quantity: 1, UnitCostMinorUnits: 9999},
	}

	cost, err := appprocurement.AllocateLandedUnitCost(lineItems, "line-a")

	require.NoError(t, err)
	require.Equal(t, int64(1150), cost) // 1000 + 100 + 50, discount ignored
}

func TestAllocateLandedUnitCost_noChargeLinesReturnsUnitCostUnchanged(t *testing.T) {
	lineItems := []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-a", LineType: procurementDomain.LineItemTypeProduct, Quantity: 3, UnitCostMinorUnits: 500},
	}

	cost, err := appprocurement.AllocateLandedUnitCost(lineItems, "line-a")

	require.NoError(t, err)
	require.Equal(t, int64(500), cost)
}

func TestAllocateLandedUnitCost_emptyLineTypeTreatedAsProduct(t *testing.T) {
	// CSV/XLSX-parsed line items leave LineType unset -- must still count
	// toward the product subtotal and receive a fair share of shipping.
	lineItems := []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-a", LineType: "", Quantity: 1, UnitCostMinorUnits: 1000},
		{ID: "line-shipping", LineType: procurementDomain.LineItemTypeShipping, Quantity: 1, UnitCostMinorUnits: 100},
	}

	cost, err := appprocurement.AllocateLandedUnitCost(lineItems, "line-a")

	require.NoError(t, err)
	require.Equal(t, int64(1100), cost)
}

func TestAllocateLandedUnitCost_targetNotOnOrderReturnsError(t *testing.T) {
	lineItems := []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-a", LineType: procurementDomain.LineItemTypeProduct, Quantity: 1, UnitCostMinorUnits: 1000},
	}

	_, err := appprocurement.AllocateLandedUnitCost(lineItems, "does-not-exist")

	require.ErrorIs(t, err, appprocurement.ErrLineItemNotOnPurchaseOrder)
}
