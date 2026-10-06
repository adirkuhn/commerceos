package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestCreatePurchaseOrder_setsFields(t *testing.T) {
	repo := &stubPurchaseOrderRepo{}
	uc := appprocurement.NewCreatePurchaseOrderUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	po, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderInput{
		SupplierID: strPtr("supplier-1"), WarehouseID: strPtr("wh-1"), CurrencyCode: "USD", Notes: "urgent",
	})

	require.NoError(t, err)
	require.Equal(t, "supplier-1", *po.SupplierID)
	require.Equal(t, "wh-1", *po.WarehouseID)
	require.Equal(t, procurementDomain.PurchaseOrderStatusDraft, po.Status, "always starts as draft")
	require.True(t, repo.createCalled)
}

func TestCreatePurchaseOrder_allowsNilWarehouse(t *testing.T) {
	repo := &stubPurchaseOrderRepo{}
	uc := appprocurement.NewCreatePurchaseOrderUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	po, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderInput{
		SupplierID: strPtr("supplier-1"), CurrencyCode: "USD",
	})

	require.NoError(t, err)
	assert.Nil(t, po.WarehouseID, "warehouse is assigned later, at first booking/receiving, not required up front")
}

func TestCreatePurchaseOrder_startsNeedsReviewWhenSupplierOmitted(t *testing.T) {
	repo := &stubPurchaseOrderRepo{}
	uc := appprocurement.NewCreatePurchaseOrderUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	po, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderInput{WarehouseID: strPtr("wh-1"), CurrencyCode: "USD"})

	require.NoError(t, err)
	assert.Nil(t, po.SupplierID)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusNeedsReview, po.Status)
}

func TestCreatePurchaseOrder_propagatesRepoError(t *testing.T) {
	repo := &stubPurchaseOrderRepo{createErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewCreatePurchaseOrderUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderInput{SupplierID: strPtr("supplier-1"), WarehouseID: strPtr("wh-1")})

	require.Error(t, err)
}
