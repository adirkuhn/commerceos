package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestUpdatePurchaseOrderLineItem_updatesFields(t *testing.T) {
	repo := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", Quantity: 10}}
	uc := appprocurement.NewUpdatePurchaseOrderLineItemUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	li, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderLineItemInput{
		ID: "line-item-1", RawDescription: "Widget v2", Quantity: 20, UnitCostMinorUnits: 600,
	})

	require.NoError(t, err)
	require.Equal(t, int64(20), li.Quantity)
	require.Equal(t, "Widget v2", li.RawDescription)
}

func TestUpdatePurchaseOrderLineItem_confirmsLineTypeOnExplicitSave(t *testing.T) {
	repo := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{
		ID: "line-item-1", Quantity: 1, LineType: procurementDomain.LineItemTypeShipping, LineTypeConfirmed: false,
	}}
	uc := appprocurement.NewUpdatePurchaseOrderLineItemUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	li, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderLineItemInput{
		ID: "line-item-1", RawDescription: "Freight", Quantity: 1, UnitCostMinorUnits: 1500,
		LineType: procurementDomain.LineItemTypeShipping,
	})

	require.NoError(t, err)
	require.Equal(t, procurementDomain.LineItemTypeShipping, li.LineType)
	require.True(t, li.LineTypeConfirmed, "an explicit save through this form confirms the AI's guess (or the human's correction)")
}

func TestUpdatePurchaseOrderLineItem_rejectsNonPositiveQuantity(t *testing.T) {
	repo := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-item-1"}}
	uc := appprocurement.NewUpdatePurchaseOrderLineItemUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderLineItemInput{ID: "line-item-1", Quantity: -1})

	require.ErrorIs(t, err, procurementDomain.ErrInvalidQuantity)
}

func TestUpdatePurchaseOrderLineItem_propagatesNotFound(t *testing.T) {
	repo := &stubLineItemRepo{findErr: procurementDomain.ErrLineItemNotFound}
	uc := appprocurement.NewUpdatePurchaseOrderLineItemUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderLineItemInput{ID: "missing", Quantity: 1})

	require.ErrorIs(t, err, procurementDomain.ErrLineItemNotFound)
}
