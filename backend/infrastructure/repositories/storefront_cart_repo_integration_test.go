//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func seedInternalSalesChannel(t *testing.T, db *gorm.DB, ctx context.Context) string {
	t.Helper()
	repo := repositories.NewSalesChannelRepo(db)
	var created channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "B2C Store", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	return created.ID
}

// TestStorefrontCartRepo_Integration_CreateFindUpdate proves the JSONB
// items column round-trips through real Postgres (not just sqlmock) --
// same reasoning as every other JSONB-backed repo's integration coverage
// in this file.
func TestStorefrontCartRepo_Integration_CreateFindUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	channelID := seedInternalSalesChannel(t, db, ctx)
	repo := repositories.NewStorefrontCartRepo(db)

	var created storefrontDomain.Cart
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = storefrontDomain.Cart{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: channelID, Token: "test-token-1",
			CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *storefrontDomain.Cart
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByToken(ctx, "test-token-1")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Empty(t, found.Items)
	require.Nil(t, found.CheckedOutAt)

	found.Items = []storefrontDomain.CartItem{
		{SKUID: "sku-1", SKUCode: "SKU-1", ProductName: "Widget", Quantity: 2, PriceMinorUnits: 500, CurrencyCode: "EUR"},
	}
	code := "SUMMER10"
	found.CouponCode = &code
	found.UpdatedAt = time.Now()
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Update(ctx, found)
	})
	require.NoError(t, err)

	var updated *storefrontDomain.Cart
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		updated, err = repo.FindByToken(ctx, "test-token-1")
		return err
	})
	require.NoError(t, err)
	require.Len(t, updated.Items, 1)
	require.Equal(t, "SKU-1", updated.Items[0].SKUCode)
	require.Equal(t, int64(2), updated.Items[0].Quantity)
	require.NotNil(t, updated.CouponCode)
	require.Equal(t, "SUMMER10", *updated.CouponCode)

	// Removing the coupon (nil) must persist as NULL, not linger as the
	// old code.
	updated.CouponCode = nil
	updated.UpdatedAt = time.Now()
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Update(ctx, updated)
	})
	require.NoError(t, err)

	var cleared *storefrontDomain.Cart
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		cleared, err = repo.FindByToken(ctx, "test-token-1")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, cleared.CouponCode)
}

// TestStorefrontCartRepo_Integration_FindByTokenNotFound proves a missing
// token returns (nil, nil), not an error -- see CartRepository.FindByToken's
// doc comment.
func TestStorefrontCartRepo_Integration_FindByTokenNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewStorefrontCartRepo(db)

	var found *storefrontDomain.Cart
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByToken(ctx, "does-not-exist")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}

// TestStorefrontCartRepo_Integration_TenantIsolation proves a cart created
// under one tenant is invisible to another -- RLS enforcement, not just an
// application-level WHERE clause. Must run as commerceos_app (see
// newAppRoleDB's doc comment) -- the superuser connection newIntegrationDB
// returns bypasses RLS unconditionally regardless of FORCE ROW LEVEL
// SECURITY, so this test would pass even with RLS completely broken if it
// used that connection instead.
func TestStorefrontCartRepo_Integration_TenantIsolation(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	channelID := seedInternalSalesChannel(t, superDB, ctxA)
	repo := repositories.NewStorefrontCartRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		now := time.Now()
		cart := storefrontDomain.Cart{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: channelID, Token: "shared-token",
			CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &cart)
	})
	require.NoError(t, err)

	var foundFromB *storefrontDomain.Cart
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		foundFromB, err = repo.FindByToken(ctx, "shared-token")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, foundFromB, "tenant B must never see tenant A's cart")
}
