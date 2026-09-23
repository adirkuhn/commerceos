//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestNotificationPreferencesRepo_NilBeforeSave(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewNotificationPreferencesRepo(db)
	_, ctx := newTestTenant()

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		p, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.Nil(t, p)
		return nil
	}))
}

func TestNotificationPreferencesRepo_UpsertThenGet(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewNotificationPreferencesRepo(db)
	tenantID, ctx := newTestTenant()
	p, err := notificationsDomain.NewPreferences(tenantID, notificationsDomain.PreferencesInput{
		Categories: map[notificationsDomain.Category]notificationsDomain.ChannelPrefs{
			notificationsDomain.CategoryImports: {InApp: false, Email: true},
		},
		EmailRecipients: []string{"ops@example.nl"},
	}, time.Now().UTC())
	require.NoError(t, err)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		require.NoError(t, repo.Upsert(ctx, p))
		p.EmailRecipients = []string{"boss@example.nl"}
		require.NoError(t, repo.Upsert(ctx, p))
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, []string{"boss@example.nl"}, got.EmailRecipients)
		assert.Equal(t, notificationsDomain.ChannelPrefs{InApp: false, Email: true}, got.For(notificationsDomain.CategoryImports))
		assert.Equal(t, notificationsDomain.ChannelPrefs{InApp: true}, got.For(notificationsDomain.CategoryChannelOffers))
		return nil
	}))
}

func TestNotificationPreferencesRepo_TenantIsolation(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	repo := repositories.NewNotificationPreferencesRepo(db)
	tenantA, ctxA := newTestTenant()
	_, ctxB := newTestTenant()

	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, notificationsDomain.DefaultPreferences(tenantA))
	}))
	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.Nil(t, got)
		return nil
	}))
}
