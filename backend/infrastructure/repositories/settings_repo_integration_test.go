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

func TestTenantSettingsRepo_Integration_GetReturnsZeroValueWhenNoRowSaved(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewTenantSettingsRepo(db)

	var found *settingsDomain.TenantSettings
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	})
	require.NoError(t, err)
	require.False(t, found.InventorySyncRulesEnabled)
	require.Equal(t, 0, found.BatchExpiryThresholdDays)
}

func TestTenantSettingsRepo_Integration_UpsertRoundTripsInventorySyncRuleFields(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewTenantSettingsRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &settingsDomain.TenantSettings{
			TenantID: tenantID, DefaultCurrency: "EUR",
			InventorySyncRulesEnabled: true, BatchExpiryThresholdDays: 30,
			UpdatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	var found *settingsDomain.TenantSettings
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	})
	require.NoError(t, err)
	require.True(t, found.InventorySyncRulesEnabled)
	require.Equal(t, 30, found.BatchExpiryThresholdDays)

	// Upsert again with the flag turned back off must persist false, not
	// silently keep the prior true (OnConflict DoUpdates must list the column).
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &settingsDomain.TenantSettings{
			TenantID: tenantID, DefaultCurrency: "EUR",
			InventorySyncRulesEnabled: false, BatchExpiryThresholdDays: 0,
			UpdatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	})
	require.NoError(t, err)
	require.False(t, found.InventorySyncRulesEnabled)
	require.Equal(t, 0, found.BatchExpiryThresholdDays)
}
