//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestScannerPairingCodeRepo_Integration_CreateFindClaim(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewScannerPairingCodeRepo(db)

	var created iamDomain.ScannerPairingCode
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = iamDomain.ScannerPairingCode{
			TenantID: tenantID, Code: "123456", UserID: "user-1", Role: "customer_owner",
			Status: iamDomain.ScannerPairingCodeStatusPending, ExpiresAt: time.Now().Add(5 * time.Minute), CreatedAt: time.Now(),
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *iamDomain.ScannerPairingCode
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, iamDomain.ScannerPairingCodeStatusPending, found.Status)

	adminCtx := tenant.WithSuperAdmin(context.Background())
	var pending *iamDomain.ScannerPairingCode
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = repo.FindPendingByCode(ctx, "123456")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, pending)
	require.Equal(t, created.ID, pending.ID)

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return repo.MarkClaimed(ctx, created.ID, time.Now())
	}))

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, iamDomain.ScannerPairingCodeStatusClaimed, found.Status)
	require.NotNil(t, found.ClaimedAt)

	// Once claimed, the code is no longer pending -- a second claim attempt
	// with the same text must not find it (see the partial unique index
	// scoping uniqueness to pending codes only).
	var goneNowClaimed *iamDomain.ScannerPairingCode
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		goneNowClaimed, err = repo.FindPendingByCode(ctx, "123456")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, goneNowClaimed)
}

func TestScannerPairingCodeRepo_Integration_FindPendingByToken(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewScannerPairingCodeRepo(db)

	var created iamDomain.ScannerPairingCode
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = iamDomain.ScannerPairingCode{
			TenantID: tenantID, Code: "123456", Token: "opaque-token-abc", UserID: "user-1", Role: "customer_owner",
			Status: iamDomain.ScannerPairingCodeStatusPending, ExpiresAt: time.Now().Add(5 * time.Minute), CreatedAt: time.Now(),
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)

	adminCtx := tenant.WithSuperAdmin(context.Background())
	var pending *iamDomain.ScannerPairingCode
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = repo.FindPendingByToken(ctx, "opaque-token-abc")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, pending)
	require.Equal(t, created.ID, pending.ID)

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return repo.MarkClaimed(ctx, created.ID, time.Now())
	}))

	// Once claimed, the token is no longer pending -- same partial-unique-
	// index reasoning as the code path.
	var goneNowClaimed *iamDomain.ScannerPairingCode
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		goneNowClaimed, err = repo.FindPendingByToken(ctx, "opaque-token-abc")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, goneNowClaimed)
}

func TestScannerPairingCodeRepo_Integration_FindPendingByCodeExpired(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewScannerPairingCodeRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		code := iamDomain.ScannerPairingCode{
			TenantID: tenantID, Code: "999999", UserID: "user-1", Role: "customer_owner",
			Status: iamDomain.ScannerPairingCodeStatusPending, ExpiresAt: time.Now().Add(-time.Minute), CreatedAt: time.Now(),
		}
		return repo.Create(ctx, &code)
	})
	require.NoError(t, err)

	adminCtx := tenant.WithSuperAdmin(context.Background())
	var found *iamDomain.ScannerPairingCode
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindPendingByCode(ctx, "999999")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found, "an expired code must not be returned even though its status is still pending")
}

func TestScannerPairingCodeRepo_Integration_NotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewScannerPairingCodeRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, iamDomain.ErrPairingCodeNotFound)
}
