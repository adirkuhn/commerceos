package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestAddPurchaseOrderLineItem_setsFields(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	skuID := "sku-1"
	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", SKUID: &skuID, RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500,
	})

	require.NoError(t, err)
	require.Equal(t, "po-1", li.PurchaseOrderID)
	require.Equal(t, "sku-1", *li.SKUID)
	require.Equal(t, int64(10), li.Quantity)
	require.True(t, repo.createCalled)
}

func TestAddPurchaseOrderLineItem_manuallyAddedItemIsConfirmedImmediately(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Shipping", Quantity: 1, UnitCostMinorUnits: 1500,
		LineType: procurementDomain.LineItemTypeShipping,
	})

	require.NoError(t, err)
	require.Equal(t, procurementDomain.LineItemTypeShipping, li.LineType)
	require.True(t, li.LineTypeConfirmed, "a human typed this in directly -- no ambiguity to review")
}

func TestAddPurchaseOrderLineItem_rejectsNonPositiveQuantity(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", Quantity: 0,
	})

	require.ErrorIs(t, err, procurementDomain.ErrInvalidQuantity)
	require.False(t, repo.createCalled)
}

func TestAddPurchaseOrderLineItem_propagatesRepoError(t *testing.T) {
	repo := &stubLineItemRepo{createErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{PurchaseOrderID: "po-1", Quantity: 1})

	require.Error(t, err)
}
