//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestShippingPreferencesRepo_Integration_GetBeforeAnySaveIsNil(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewShippingPreferencesRepo(db)

	var found *shippingDomain.ShippingPreferences
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	})

	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestShippingPreferencesRepo_Integration_UpsertRoundTripsAndUpdatesInPlace(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	first := seedMyParcelConnection(t, db, ctx)
	second := seedConnectionOfProvider(t, db, ctx, "postnl")
	repo := repositories.NewShippingPreferencesRepo(db)

	save := func(id *string) {
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			return repo.Upsert(ctx, &shippingDomain.ShippingPreferences{TenantID: tenant.IDFromContext(ctx), DefaultConnectionID: id, UpdatedAt: time.Now()})
		})
		require.NoError(t, err)
	}
	get := func() *shippingDomain.ShippingPreferences {
		var found *shippingDomain.ShippingPreferences
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = repo.Get(ctx)
			return err
		})
		require.NoError(t, err)
		require.NotNil(t, found)
		return found
	}

	save(&first)
	require.NotNil(t, get().DefaultConnectionID)
	assert.Equal(t, first, *get().DefaultConnectionID)

	save(&second)
	assert.Equal(t, second, *get().DefaultConnectionID)

	save(nil)
	assert.Nil(t, get().DefaultConnectionID)
}

func TestShippingPreferencesRepo_Integration_OtherTenantCannotSeeIt(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	conn := seedMyParcelConnection(t, db, ctxA)
	repo := repositories.NewShippingPreferencesRepo(db)
	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &shippingDomain.ShippingPreferences{TenantID: tenant.IDFromContext(ctx), DefaultConnectionID: &conn, UpdatedAt: time.Now()})
	}))

	var found *shippingDomain.ShippingPreferences
	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	}))

	assert.Nil(t, found)
}

func TestShippingPreferencesRepo_Integration_DeletingTheConnectionClearsTheChoice(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	conn := seedMyParcelConnection(t, db, ctx)
	repo := repositories.NewShippingPreferencesRepo(db)
	connRepo := repositories.NewConnectionRepo(db)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &shippingDomain.ShippingPreferences{TenantID: tenant.IDFromContext(ctx), DefaultConnectionID: &conn, UpdatedAt: time.Now()})
	}))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error { return connRepo.Delete(ctx, conn) }))

	var found *shippingDomain.ShippingPreferences
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	}))
	require.NotNil(t, found)
	assert.Nil(t, found.DefaultConnectionID)
}

func seedConnectionOfProvider(t *testing.T, db *gorm.DB, ctx context.Context, provider integrationsDomain.Provider) string {
	t.Helper()
	conn := &integrationsDomain.Connection{Provider: provider, ConsumerSecretEncrypted: "enc", Status: integrationsDomain.StatusConnected}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		conn.TenantID = tenant.IDFromContext(ctx)
		conn.CreatedAt, conn.UpdatedAt = time.Now(), time.Now()
		return repositories.NewConnectionRepo(db).Create(ctx, conn)
	}))
	return conn.ID
}

func TestShippingPreferencesRepo_Integration_RoundTripsAddresses(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewShippingPreferencesRepo(db)
	sender := &shippingDomain.ShippingAddress{Company: "Cantinho", Street: "Damrak", Number: "1", NumberSuffix: "A", PostalCode: "1012AB", City: "Amsterdam", CC: "NL", Email: "shop@example.com"}
	ret := &shippingDomain.ShippingAddress{Company: "Cantinho Returns", Street: "Kade", Number: "9", PostalCode: "3011AA", City: "Rotterdam", CC: "NL"}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &shippingDomain.ShippingPreferences{TenantID: tenant.IDFromContext(ctx), SenderAddress: sender, ReturnAddress: ret, UpdatedAt: time.Now()})
	}))

	var found *shippingDomain.ShippingPreferences
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	}))
	require.NotNil(t, found.SenderAddress)
	require.NotNil(t, found.ReturnAddress)
	assert.Equal(t, *sender, *found.SenderAddress)
	assert.Equal(t, *ret, *found.ReturnAddress)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &shippingDomain.ShippingPreferences{TenantID: tenant.IDFromContext(ctx), SenderAddress: sender, UpdatedAt: time.Now()})
	}))
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.Get(ctx)
		return err
	}))
	assert.Nil(t, found.ReturnAddress, "clearing the return address stores NULL")
}
