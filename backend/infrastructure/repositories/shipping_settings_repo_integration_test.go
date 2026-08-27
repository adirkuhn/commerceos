//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestShippingConnectionSettingsRepo_Integration_UpsertRoundTripsMyParcelFields
// is a regression test for a real production error: MyParcelAccountID/
// MyParcelShopID had no explicit gorm column tag, so GORM's default namer
// derived "my_parcel_account_id" while migration 096/111 actually named the
// column "myparcel_account_id" (matching this codebase's other
// MyParcel-prefixed columns) -- Upsert failed with "column
// my_parcel_account_id does not exist" (SQLSTATE 42703), invisible to
// sqlmock-backed unit tests since they don't validate against a real
// schema.
func TestShippingConnectionSettingsRepo_Integration_UpsertRoundTripsMyParcelFields(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connectionID := seedMyParcelConnection(t, db, ctx)
	repo := repositories.NewShippingConnectionSettingsRepo(db)

	accountID, shopID := 111, 222
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		return repo.Upsert(ctx, &shippingDomain.ShippingConnectionSettings{
			ConnectionID: connectionID, TenantID: tenant.IDFromContext(ctx),
			DefaultCarrier: shippingDomain.CarrierPostNL, DefaultPackageType: shippingDomain.PackageTypePackage,
			DefaultLabelFormat: shippingDomain.LabelFormatPDF,
			InsureByDefault:    true,
			MyParcelAccountID:  &accountID, MyParcelShopID: &shopID, AutoPrintEnabled: true,
			CreatedAt: now, UpdatedAt: now,
		})
	})
	require.NoError(t, err)

	var found *shippingDomain.ShippingConnectionSettings
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByConnectionID(ctx, connectionID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.MyParcelAccountID)
	require.NotNil(t, found.MyParcelShopID)
	assert.Equal(t, 111, *found.MyParcelAccountID)
	assert.Equal(t, 222, *found.MyParcelShopID)
	assert.True(t, found.AutoPrintEnabled)
	assert.True(t, found.InsureByDefault)
}
