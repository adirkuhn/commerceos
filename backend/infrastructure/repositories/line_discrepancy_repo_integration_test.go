//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestLineDiscrepancyRepo_Integration_CreateSaveListAndFind(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	repo := repositories.NewLineDiscrepancyRepo(db)
	detected := time.Now().UTC().Truncate(time.Second)

	missing := omsDomain.LineDiscrepancy{
		TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, Kind: omsDomain.LineDiscrepancyMissing,
		ItemKey: "sku:7896015976606", SKUCode: "7896015976606", Name: "Mistura de Pão de Queijo - Pinduca - 500g",
		ChannelQuantity: 1, DetectedAt: detected,
	}
	total := omsDomain.TotalDiscrepancy(3641, 3293)
	total.TenantID, total.SalesOrderID, total.DetectedAt = tenant.IDFromContext(ctx), orderID, detected.Add(-time.Minute)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := repo.Create(ctx, &missing); err != nil {
			return err
		}
		return repo.Create(ctx, &total)
	}))
	require.NotEmpty(t, missing.ID)

	var byOrder []omsDomain.LineDiscrepancy
	var open []omsDomain.OpenLineDiscrepancy
	var count int64
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if byOrder, err = repo.ListBySalesOrder(ctx, orderID); err != nil {
			return err
		}
		open, count, err = repo.ListOpen(ctx, 1, 10)
		return err
	}))
	require.Len(t, byOrder, 2)
	assert.Equal(t, int64(2), count)
	require.Len(t, open, 2)
	assert.Equal(t, missing.ID, open[0].ID, "newest first")
	assert.Equal(t, orderID, open[0].Order.ID, "carries its order")
	assert.Equal(t, "Mistura de Pão de Queijo - Pinduca - 500g", open[0].Name)
	assert.Equal(t, int64(3641), open[1].ChannelTotalMinorUnits)

	require.NoError(t, missing.Dismiss(detected))
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error { return repo.Save(ctx, &missing) }))

	var found *omsDomain.LineDiscrepancy
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if found, err = repo.FindByID(ctx, missing.ID); err != nil {
			return err
		}
		open, count, err = repo.ListOpen(ctx, 1, 10)
		return err
	}))
	require.NotNil(t, found.DismissedAt)
	assert.Equal(t, int64(1), count, "dismissed is no longer open")
	require.Len(t, open, 1)
	assert.Equal(t, total.ID, open[0].ID)
}

func TestLineDiscrepancyRepo_Integration_FindUnknown(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewLineDiscrepancyRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, idgen.NewV4())
		return err
	})

	assert.ErrorIs(t, err, omsDomain.ErrLineDiscrepancyNotFound)
}

func TestLineDiscrepancyRepo_Integration_RLSHidesOtherTenants(t *testing.T) {
	db, superDSN := newIntegrationDBAndDSN(t)
	appDB := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	_, wh := seedProductSKUWarehouse(t, db, ctxA)
	orderID := seedSalesOrderForPickingDB(t, db, ctxA, wh)
	d := omsDomain.LineDiscrepancy{
		TenantID: tenant.IDFromContext(ctxA), SalesOrderID: orderID, Kind: omsDomain.LineDiscrepancyMissing,
		ItemKey: "sku:a", SKUCode: "A", ChannelQuantity: 1, DetectedAt: time.Now(),
	}
	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return repositories.NewLineDiscrepancyRepo(db).Create(ctx, &d)
	}))

	var count int64
	require.NoError(t, tenant.WithTenant(ctxB, appDB, func(ctx context.Context) error {
		var err error
		_, count, err = repositories.NewLineDiscrepancyRepo(appDB).ListOpen(ctx, 1, 10)
		return err
	}))

	assert.Zero(t, count)
}
