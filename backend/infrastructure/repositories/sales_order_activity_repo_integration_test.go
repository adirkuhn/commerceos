//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSalesOrderActivityRepo_Integration_CreateAndListBySalesOrderAndByUser(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	repo := repositories.NewSalesOrderActivityRepo(db)
	userID := uuid.NewString()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		a := &omsDomain.SalesOrderActivity{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, UserID: userID,
			ActivityType: "myparcel_shipment_created", Summary: "Shipment created", Metadata: []byte(`{"barcode":"3SMYPA1"}`),
		}
		return repo.Create(ctx, a)
	})
	require.NoError(t, err)

	var bySalesOrder []omsDomain.SalesOrderActivity
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		bySalesOrder, err = repo.ListBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, bySalesOrder, 1)
	require.Equal(t, "myparcel_shipment_created", bySalesOrder[0].ActivityType)
	require.JSONEq(t, `{"barcode":"3SMYPA1"}`, string(bySalesOrder[0].Metadata))

	var byUser []omsDomain.SalesOrderActivity
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byUser, err = repo.ListByUser(ctx, userID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, byUser, 1)
	require.Equal(t, orderID, byUser[0].SalesOrderID)
}

func TestSalesOrderActivityRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)
	orderID := seedSalesOrderForPickingDB(t, superDB, ctxA, warehouseID)
	repo := repositories.NewSalesOrderActivityRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		a := &omsDomain.SalesOrderActivity{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, UserID: uuid.NewString(),
			ActivityType: "myparcel_shipment_created", Summary: "Shipment created", Metadata: []byte(`{}`),
		}
		return repo.Create(ctx, a)
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	var bySalesOrder []omsDomain.SalesOrderActivity
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		bySalesOrder, err = repo.ListBySalesOrder(ctx, orderID)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, bySalesOrder, "tenant B must not see tenant A's order activity")
}
