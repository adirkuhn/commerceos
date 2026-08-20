package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/require"
)

func TestGetLandedUnitCost_computesFromTheLineItemsSharedPurchaseOrder(t *testing.T) {
	db := newMockGormDB(t)
	lineItemRepo := &stubLineItemRepo{
		findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-1", PurchaseOrderID: "po-1"},
		byPO: []procurementDomain.PurchaseOrderLineItem{
			{ID: "line-1", Quantity: 10, UnitCostMinorUnits: 800, LineType: procurementDomain.LineItemTypeProduct},
			{ID: "line-shipping", Quantity: 1, UnitCostMinorUnits: 1000, LineType: procurementDomain.LineItemTypeShipping},
		},
	}
	uc := appprocurement.NewGetLandedUnitCostUseCase(db, lineItemRepo)

	cost, err := uc.Execute(context.Background(), "line-1")

	require.NoError(t, err)
	require.Equal(t, int64(900), cost) // 800 + all 1000 of shipping / 10 units (only product line)
}

func TestGetLandedUnitCost_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	lineItemRepo := &stubLineItemRepo{findErr: procurementDomain.ErrLineItemNotFound}
	uc := appprocurement.NewGetLandedUnitCostUseCase(db, lineItemRepo)

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
}
