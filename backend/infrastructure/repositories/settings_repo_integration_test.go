//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
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
	require.Equal(t, settingsDomain.PickingModeScan, found.PickingMode, "a never-saved tenant picks by scanning")
	require.Equal(t, settingsDomain.DateFormatDayMonthYear, found.DateFormat, "a never-saved tenant reads dates as DD/MM/YYYY")
}

func TestTenantSettingsRepo_Integration_UpsertRoundTripsDateFormat(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewTenantSettingsRepo(db)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &settingsDomain.TenantSettings{TenantID: tenantID, DateFormat: settingsDomain.DateFormatMonthDayYear, UpdatedAt: time.Now()})
	}))

	var found *settingsDomain.TenantSettings
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	}))
	require.Equal(t, settingsDomain.DateFormatMonthDayYear, found.DateFormat)
}

func TestTenantSettingsRepo_Integration_UpsertRoundTripsPickingMode(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewTenantSettingsRepo(db)
	get := func() settingsDomain.PickingMode {
		var found *settingsDomain.TenantSettings
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = repo.Get(ctx)
			return err
		}))
		return found.PickingMode
	}
	upsert := func(mode settingsDomain.PickingMode) error {
		return tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			return repo.Upsert(ctx, &settingsDomain.TenantSettings{TenantID: tenantID, PickingMode: mode, UpdatedAt: time.Now()})
		})
	}

	require.NoError(t, upsert(settingsDomain.PickingModeList))
	require.Equal(t, settingsDomain.PickingModeList, get())

	require.NoError(t, upsert(""), "an unset mode is stored as the default")
	require.Equal(t, settingsDomain.PickingModeScan, get())

	require.ErrorIs(t, upsert("barcode"), settingsDomain.ErrInvalidPickingMode)
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

func TestTenantSettingsRepo_Integration_UpsertRoundTripsLanguage(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewTenantSettingsRepo(db)
	save := func(l locale.Locale) locale.Locale {
		var found *settingsDomain.TenantSettings
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			if err := repo.Upsert(ctx, &settingsDomain.TenantSettings{TenantID: tenantID, Language: l, UpdatedAt: time.Now()}); err != nil {
				return err
			}
			var err error
			found, err = repo.Get(ctx)
			return err
		}))
		return found.Language
	}

	require.Equal(t, locale.Portuguese, save(locale.Portuguese))
	require.Equal(t, locale.English, save(""), "unset is stored as English")
}

func TestTenantSettingsRepo_Integration_UpsertRoundTripsExpiryAlert(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewTenantSettingsRepo(db)
	get := func() settingsDomain.ExpiryAlert {
		var found *settingsDomain.TenantSettings
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = repo.Get(ctx)
			return err
		}))
		return found.ExpiryAlert
	}

	require.Equal(t, settingsDomain.ExpiryAlert{Windows: []int{}}, get(), "a never-saved tenant has the alert off")

	alert := settingsDomain.ExpiryAlert{Enabled: true, Windows: []int{90, 15, 7}}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &settingsDomain.TenantSettings{TenantID: tenantID, ExpiryAlert: alert, UpdatedAt: time.Now()})
	}))
	require.Equal(t, alert, get())
}
