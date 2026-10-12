//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestMessageCenterSetupRepo_Integration_claimsATenantOnce(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewMessageCenterSetupRepo(db)
	claim := func(ctx context.Context) bool {
		var ok bool
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			ok, err = repo.Claim(ctx, time.Now())
			return err
		}))
		return ok
	}

	require.True(t, claim(ctx))
	require.False(t, claim(ctx), "second claim seeds nothing")

	_, otherCtx := newTestTenant()
	require.True(t, claim(otherCtx), "claims are per tenant")
}

func TestMessageCenterSetupRepo_Integration_listsUnseededTenants(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewMessageCenterSetupRepo(db)
	settings := repositories.NewTenantSettingsRepo(db)
	newTenant := func() (string, context.Context) {
		id, ctx := newTestTenant()
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			return settings.Upsert(ctx, &settingsDomain.TenantSettings{TenantID: id, UpdatedAt: time.Now()})
		}))
		return id, ctx
	}
	freshID, _ := newTenant()
	doneID, doneCtx := newTenant()
	require.NoError(t, tenant.WithTenant(doneCtx, db, func(ctx context.Context) error {
		_, err := repo.Claim(ctx, time.Now())
		return err
	}))

	var ids []string
	require.NoError(t, tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		var err error
		ids, err = repo.ListUnseededTenants(ctx)
		return err
	}))

	require.Contains(t, ids, freshID)
	require.NotContains(t, ids, doneID)
}
