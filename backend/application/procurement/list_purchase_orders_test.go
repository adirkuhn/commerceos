package procurement_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestListPurchaseOrders_returnsRepoResult(t *testing.T) {
	repo := &stubPurchaseOrderRepo{purchaseOrders: []procurementDomain.PurchaseOrder{{ID: "po-1"}}}
	uc := appprocurement.NewListPurchaseOrdersUseCase(newMockGormDB(t), repo, &stubLineItemRepo{})

	pos, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, pos, 1)
	require.Equal(t, "po-1", pos[0].ID)
}

// TestListPurchaseOrders_attachesTotalMinorUnitsFromLineItemSum is the "PO
// total on the main list" feature: the list endpoint doesn't load full line
// items per PO (would be an N+1 fetch), so it asks the line item repo for
// one aggregate sum per PO instead and attaches it to TotalMinorUnits.
func TestListPurchaseOrders_attachesTotalMinorUnitsFromLineItemSum(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{purchaseOrders: []procurementDomain.PurchaseOrder{{ID: "po-1"}, {ID: "po-2"}}}
	lineItemRepo := &stubLineItemRepo{sumByPO: map[string]int64{"po-1": 1300, "po-2": 0}}
	uc := appprocurement.NewListPurchaseOrdersUseCase(newMockGormDB(t), poRepo, lineItemRepo)

	pos, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, pos, 2)
	require.Equal(t, int64(1300), pos[0].TotalMinorUnits)
	require.Equal(t, int64(0), pos[1].TotalMinorUnits)
}

func TestListPurchaseOrders_propagatesSumError(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{purchaseOrders: []procurementDomain.PurchaseOrder{{ID: "po-1"}}}
	lineItemRepo := &stubLineItemRepo{sumErr: errors.New("db down")}
	uc := appprocurement.NewListPurchaseOrdersUseCase(newMockGormDB(t), poRepo, lineItemRepo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
