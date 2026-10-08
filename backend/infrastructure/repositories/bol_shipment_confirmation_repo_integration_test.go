//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func seedBolConfirmation(t *testing.T, db *gorm.DB, ctx context.Context, orderID string, status channelsDomain.BolConfirmationStatus, createdAt time.Time) channelsDomain.BolShipmentConfirmation {
	t.Helper()
	repo := repositories.NewBolShipmentConfirmationRepo(db)
	c := channelsDomain.BolShipmentConfirmation{
		TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, ConnectionID: idgen.NewV4(), ShipmentID: idgen.NewV4(),
		ExternalOrderID: "C000HLRF87", ProcessStatusID: "ps-" + string(status), TransporterCode: "TNT",
		TrackAndTrace: "3SMYPA1", Status: status, CreatedAt: createdAt,
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error { return repo.Create(ctx, &c) }))
	require.NotEmpty(t, c.ID)
	return c
}

func TestBolShipmentConfirmationRepo_Integration_CreateAndListBySalesOrder(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	older := seedBolConfirmation(t, db, ctx, orderID, channelsDomain.BolConfirmationFailure, time.Now().Add(-time.Hour))
	newer := seedBolConfirmation(t, db, ctx, orderID, channelsDomain.BolConfirmationPending, time.Now())
	repo := repositories.NewBolShipmentConfirmationRepo(db)

	var got []channelsDomain.BolShipmentConfirmation
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		got, err = repo.ListBySalesOrder(ctx, orderID)
		return err
	}))

	require.Len(t, got, 2)
	assert.Equal(t, newer.ID, got[0].ID, "newest first")
	assert.Equal(t, older.ID, got[1].ID)
	assert.Equal(t, "TNT", got[0].TransporterCode)
	assert.Equal(t, "3SMYPA1", got[0].TrackAndTrace)
	assert.Equal(t, "C000HLRF87", got[0].ExternalOrderID)
	assert.Equal(t, channelsDomain.BolConfirmationPending, got[0].Status)
	assert.Nil(t, got[0].ResolvedAt)
}

func TestBolShipmentConfirmationRepo_Integration_ListPendingAcrossTenantsAndSaveResolution(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	_, whA := seedProductSKUWarehouse(t, db, ctxA)
	_, whB := seedProductSKUWarehouse(t, db, ctxB)
	orderA := seedSalesOrderForPickingDB(t, db, ctxA, whA)
	orderB := seedSalesOrderForPickingDB(t, db, ctxB, whB)
	pendingA := seedBolConfirmation(t, db, ctxA, orderA, channelsDomain.BolConfirmationPending, time.Now().Add(-2*time.Minute))
	pendingB := seedBolConfirmation(t, db, ctxB, orderB, channelsDomain.BolConfirmationPending, time.Now().Add(-time.Minute))
	seedBolConfirmation(t, db, ctxA, orderA, channelsDomain.BolConfirmationSuccess, time.Now())
	repo := repositories.NewBolShipmentConfirmationRepo(db)
	adminCtx := tenant.WithSuperAdmin(context.Background())

	var pending []channelsDomain.BolShipmentConfirmation
	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = repo.ListPending(ctx, 10)
		return err
	}))
	require.Len(t, pending, 2)
	assert.Equal(t, pendingA.ID, pending[0].ID, "oldest first")
	assert.Equal(t, pendingB.ID, pending[1].ID)
	assert.Equal(t, pendingB.TenantID, pending[1].TenantID)

	resolvedAt := time.Now().UTC().Truncate(time.Second)
	toResolve := pending[0]
	require.NoError(t, toResolve.Resolve(channelsDomain.BolConfirmationFailure, "Invalid track and trace", resolvedAt))
	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error { return repo.SaveResolution(ctx, &toResolve) }))

	var afterA []channelsDomain.BolShipmentConfirmation
	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		var err error
		afterA, err = repo.ListBySalesOrder(ctx, orderA)
		return err
	}))
	var saved channelsDomain.BolShipmentConfirmation
	for _, c := range afterA {
		if c.ID == pendingA.ID {
			saved = c
		}
	}
	assert.Equal(t, channelsDomain.BolConfirmationFailure, saved.Status)
	assert.Equal(t, "Invalid track and trace", saved.ErrorMessage)
	require.NotNil(t, saved.ResolvedAt)
	assert.True(t, resolvedAt.Equal(*saved.ResolvedAt))

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = repo.ListPending(ctx, 10)
		return err
	}))
	require.Len(t, pending, 1, "resolved confirmation is no longer pending")
	assert.Equal(t, pendingB.ID, pending[0].ID)
}

func TestBolShipmentConfirmationRepo_Integration_SaveResolution_unknownID(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewBolShipmentConfirmationRepo(db)
	c := channelsDomain.BolShipmentConfirmation{ID: idgen.NewV4(), Status: channelsDomain.BolConfirmationSuccess}

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error { return repo.SaveResolution(ctx, &c) })

	assert.ErrorIs(t, err, channelsDomain.ErrBolConfirmationNotFound)
}

func TestBolShipmentConfirmationRepo_Integration_RLSHidesOtherTenants(t *testing.T) {
	db, superDSN := newIntegrationDBAndDSN(t)
	appDB := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	_, whA := seedProductSKUWarehouse(t, db, ctxA)
	orderA := seedSalesOrderForPickingDB(t, db, ctxA, whA)
	seedBolConfirmation(t, db, ctxA, orderA, channelsDomain.BolConfirmationPending, time.Now())
	repo := repositories.NewBolShipmentConfirmationRepo(appDB)

	var got []channelsDomain.BolShipmentConfirmation
	require.NoError(t, tenant.WithTenant(ctxB, appDB, func(ctx context.Context) error {
		var err error
		got, err = repo.ListBySalesOrder(ctx, orderA)
		return err
	}))

	assert.Empty(t, got)
}
