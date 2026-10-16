//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSupplierIssueStateRepo_Integration_UpsertAndList(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierIssueStateRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	otherPOID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	ref := procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: "0b9f5a52-7a2c-4d8a-9a51-3c1f0d9e8a11"}
	claimedAt := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
	resolvedAt := time.Now().UTC().Truncate(time.Microsecond)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tenantID := tenant.IDFromContext(ctx)
		if err := repo.Upsert(ctx, &procurementDomain.SupplierIssueState{
			TenantID: tenantID, PurchaseOrderID: poID, Ref: ref,
			Status: procurementDomain.SupplierIssueStatusClaimed, Note: "emailed", UpdatedBy: "user-1", UpdatedAt: claimedAt,
		}); err != nil {
			return err
		}
		// Same ref again replaces the first row instead of adding one.
		if err := repo.Upsert(ctx, &procurementDomain.SupplierIssueState{
			TenantID: tenantID, PurchaseOrderID: poID, Ref: ref,
			Status: procurementDomain.SupplierIssueStatusResolved, Note: "credit note 42", UpdatedBy: "user-2", UpdatedAt: resolvedAt,
		}); err != nil {
			return err
		}
		return repo.Upsert(ctx, &procurementDomain.SupplierIssueState{
			TenantID: tenantID, PurchaseOrderID: otherPOID,
			Ref:    procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingShortfall, SourceID: "5e1d2c3b-4a59-4f6e-8d7c-6b5a49382716"},
			Status: procurementDomain.SupplierIssueStatusClaimed, UpdatedAt: claimedAt,
		})
	})
	require.NoError(t, err)

	var got []procurementDomain.SupplierIssueState
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		got, err = repo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Equal(t, ref, got[0].Ref)
	require.Equal(t, procurementDomain.SupplierIssueStatusResolved, got[0].Status)
	require.Equal(t, "credit note 42", got[0].Note)
	require.Equal(t, "user-2", got[0].UpdatedBy)
	require.True(t, resolvedAt.Equal(got[0].UpdatedAt))
	require.Equal(t, poID, got[0].PurchaseOrderID)
}

// Runs as commerceos_app: the superuser test connection ignores RLS.
func TestSupplierIssueStateRepo_Integration_OtherTenantSeesNothing(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	appDB := newAppRoleDB(t, dsn)
	_, ctx := newTestTenant()
	_, otherCtx := newTestTenant()
	repo := repositories.NewSupplierIssueStateRepo(appDB)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)

	err := tenant.WithTenant(ctx, appDB, func(ctx context.Context) error {
		return repo.Upsert(ctx, &procurementDomain.SupplierIssueState{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
			Ref:    procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: "0b9f5a52-7a2c-4d8a-9a51-3c1f0d9e8a11"},
			Status: procurementDomain.SupplierIssueStatusClaimed, UpdatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	var got []procurementDomain.SupplierIssueState
	err = tenant.WithTenant(otherCtx, appDB, func(ctx context.Context) error {
		var err error
		got, err = repo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, got)
}
