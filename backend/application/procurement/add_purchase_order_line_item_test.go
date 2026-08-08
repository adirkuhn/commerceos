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

func TestAddPurchaseOrderLineItem_defaultsUnitsPerPurchasedItemToOne(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), li.UnitsPerPurchasedItem)
}

// TestAddPurchaseOrderLineItem_persistsUnitsPerPurchasedItem is the "case of
// a product" edge case: a supplier's box line links to the individual unit
// SKU, with an explicit multiplier for how many stock units one purchased
// box represents (e.g. 36 for "Rapadurinha 36x25gr" linked to the 25g unit).
func TestAddPurchaseOrderLineItem_persistsUnitsPerPurchasedItem(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Rapadurinha 36x25gr", Quantity: 1, UnitCostMinorUnits: 3600,
		UnitsPerPurchasedItem: 36,
	})

	require.NoError(t, err)
	require.Equal(t, int64(36), li.UnitsPerPurchasedItem)
}

func TestAddPurchaseOrderLineItem_rejectsNegativeUnitsPerPurchasedItem(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Widget", Quantity: 1, UnitCostMinorUnits: 500,
		UnitsPerPurchasedItem: -1,
	})

	require.ErrorIs(t, err, procurementDomain.ErrInvalidUnitsPerPurchasedItem)
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

func TestAddPurchaseOrderLineItem_setsPositionAfterExistingItems(t *testing.T) {
	repo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "li-1", Position: 0}, {ID: "li-2", Position: 1},
	}}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Widget", Quantity: 1,
	})

	require.NoError(t, err)
	require.Equal(t, int64(2), li.Position)
}

func TestAddPurchaseOrderLineItem_setsPositionZeroWhenNoExistingItems(t *testing.T) {
	repo := &stubLineItemRepo{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Widget", Quantity: 1,
	})

	require.NoError(t, err)
	require.Equal(t, int64(0), li.Position)
}

func TestAddPurchaseOrderLineItem_setsPositionAfterMaxWhenAnEarlierItemWasDeleted(t *testing.T) {
	// li-1 (position 0) was deleted -- only position 2 remains. Next
	// position must be max+1 (3), not len(existing) (1), so the new item
	// still lands last instead of colliding with an existing one.
	repo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "li-3", Position: 2},
	}}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	li, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", RawDescription: "Widget", Quantity: 1,
	})

	require.NoError(t, err)
	require.Equal(t, int64(3), li.Position)
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
