//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestBarcodeAliasRepo_Integration_CreateFindListDelete(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewBarcodeAliasRepo(db)
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)

	var created pmsDomain.BarcodeAlias
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = pmsDomain.BarcodeAlias{
			TenantID: tenantID, SKUID: skuID, Code: "CASE-0001", UnitsPerScan: 12,
			CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *pmsDomain.BarcodeAlias
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByCode(ctx, "CASE-0001")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, skuID, found.SKUID)
	require.Equal(t, int64(12), found.UnitsPerScan)

	var listed []pmsDomain.BarcodeAlias
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = repo.ListBySKU(ctx, skuID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Delete(ctx, created.ID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByCode(ctx, "CASE-0001")
		return err
	})
	require.ErrorIs(t, err, pmsDomain.ErrBarcodeAliasNotFound)
}
