//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestMoneybirdSettingsRepo_Integration_RoundTripsMappings(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewMoneybirdSettingsRepo(db)
	connID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderMoneybird)

	var missing *mb.Settings
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		missing, err = repo.Get(ctx, connID)
		return err
	}))
	assert.Nil(t, missing)

	upsert := func(s mb.Settings) {
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			s.ConnectionID, s.TenantID, s.CreatedAt, s.UpdatedAt = connID, tenant.IDFromContext(ctx), time.Now(), time.Now()
			return repo.Upsert(ctx, &s)
		}))
	}
	upsert(mb.Settings{
		AdministrationID: "adm-1", AdministrationName: "Cantinho", DefaultLedgerAccountID: "led-1", AutoSync: true,
		TaxRates:       []mb.TaxRateMapping{{RateBasisPoints: 900, MoneybirdTaxRateID: "tr-9"}},
		PaymentMethods: []mb.PaymentMethodMapping{{Method: "bacs", MoneybirdPaymentMethod: "bank_transfer"}},
	})
	upsert(mb.Settings{AdministrationID: "adm-1", AdministrationName: "Cantinho", DefaultLedgerAccountID: "led-2", GuestContactID: "guest",
		TaxRates: []mb.TaxRateMapping{{RateBasisPoints: 2100, MoneybirdTaxRateID: "tr-21"}}})

	var got *mb.Settings
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		got, err = repo.Get(ctx, connID)
		return err
	}))
	require.NotNil(t, got)
	assert.Equal(t, "led-2", got.DefaultLedgerAccountID)
	assert.Equal(t, "guest", got.GuestContactID)
	assert.False(t, got.AutoSync)
	assert.Equal(t, []mb.TaxRateMapping{{RateBasisPoints: 2100, MoneybirdTaxRateID: "tr-21"}}, got.TaxRates)
	assert.Empty(t, got.PaymentMethods)
}

func TestMoneybirdDocumentRepo_Integration_OneLinkPerSource(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	_, ctx := newTestTenant()
	repo := repositories.NewMoneybirdDocumentRepo(db)
	connID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderMoneybird)
	orderID := uuid.NewString()

	save := func(d mb.Document) {
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			d.TenantID, d.ConnectionID, d.SourceType, d.SourceID, d.SyncedAt = tenant.IDFromContext(ctx), connID, mb.SourceSalesOrder, orderID, time.Now()
			return repo.Save(ctx, &d)
		}))
	}
	save(mb.Document{MoneybirdID: "inv-1", Reference: "CO-1", Status: mb.DocumentFailed, Error: "payment: boom"})
	save(mb.Document{MoneybirdID: "inv-1", Reference: "CO-1", Status: mb.DocumentSynced, PaymentRegistered: true})

	var got, other *mb.Document
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if got, err = repo.Find(ctx, mb.SourceSalesOrder, orderID); err != nil {
			return err
		}
		other, err = repo.Find(ctx, mb.SourceQuote, orderID)
		return err
	}))
	require.NotNil(t, got)
	assert.Equal(t, mb.DocumentSynced, got.Status)
	assert.True(t, got.PaymentRegistered)
	assert.Empty(t, got.Error)
	assert.Nil(t, other, "same id, other source type")

	// Another tenant can't see it.
	_, otherCtx := newTestTenant()
	var leaked *mb.Document
	require.NoError(t, tenant.WithTenant(otherCtx, newAppRoleDB(t, dsn), func(ctx context.Context) error {
		var err error
		leaked, err = repo.Find(ctx, mb.SourceSalesOrder, orderID)
		return err
	}))
	assert.Nil(t, leaked)
}

func TestMoneybirdContactRepo_Integration_RemembersCustomersContact(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewMoneybirdContactRepo(db)
	connID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderMoneybird)
	customerID := uuid.NewString()

	var before, after string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if before, err = repo.Find(ctx, connID, customerID); err != nil {
			return err
		}
		if err := repo.Save(ctx, tenant.IDFromContext(ctx), connID, customerID, "mb-c-1"); err != nil {
			return err
		}
		if err := repo.Save(ctx, tenant.IDFromContext(ctx), connID, customerID, "mb-c-2"); err != nil {
			return err
		}
		after, err = repo.Find(ctx, connID, customerID)
		return err
	}))
	assert.Empty(t, before)
	assert.Equal(t, "mb-c-2", after)
}
