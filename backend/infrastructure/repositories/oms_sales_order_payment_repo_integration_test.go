//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func splitPayments() omsDomain.SalesOrderPayments {
	return omsDomain.SalesOrderPayments{
		{Method: "Revolut", AmountMinorUnits: 300, TransactionID: "rev-1"},
		{Method: "Cash", AmountMinorUnits: 200},
		{Method: "Revolut", AmountMinorUnits: 500, TransactionID: "rev-2"},
	}
}

func storeSplitPayments(t *testing.T, db *gorm.DB, ctx context.Context, orderID string) {
	t.Helper()
	repo := repositories.NewSalesOrderPaymentRepo(db)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		order := &omsDomain.SalesOrder{ID: orderID, TenantID: tenant.IDFromContext(ctx), Payments: splitPayments()}
		return repo.CreateForSalesOrder(ctx, order)
	})
	require.NoError(t, err)
}

func listPayments(t *testing.T, db *gorm.DB, ctx context.Context, orderID string) omsDomain.SalesOrderPayments {
	t.Helper()
	repo := repositories.NewSalesOrderPaymentRepo(db)
	var payments omsDomain.SalesOrderPayments
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		payments, err = repo.ListBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	return payments
}

func TestSalesOrderPaymentRepo_Integration_StoresAndListsPaymentsInOrder(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	storeSplitPayments(t, db, ctx, orderID)

	assert.Equal(t, splitPayments(), listPayments(t, db, ctx, orderID))
}

func TestSalesOrderPaymentRepo_Integration_ListsNoPaymentsForAnOrderWithout(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)

	assert.Empty(t, listPayments(t, db, ctx, orderID))
}

func TestSalesOrderPaymentRepo_Integration_DeletingTheOrderDeletesItsPayments(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	storeSplitPayments(t, db, ctx, orderID)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repositories.NewSalesOrderRepo(db).Delete(ctx, orderID)
	})
	require.NoError(t, err)

	assert.Empty(t, listPayments(t, db, ctx, orderID))
}

func TestSalesOrderPaymentRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)
	orderID := seedSalesOrderForPickingDB(t, superDB, ctxA, warehouseID)
	storeSplitPayments(t, db, ctxA, orderID)

	_, ctxB := newTestTenant()

	assert.Empty(t, listPayments(t, db, ctxB, orderID))
}
