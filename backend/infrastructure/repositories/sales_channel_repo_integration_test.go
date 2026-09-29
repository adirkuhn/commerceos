//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// TestSalesChannelRepo_Integration_CreateAcceptsBolType is a regression
// test for a real prod bug: the sales_channels_type_check CHECK constraint
// (migration 037) was never widened to 'bol' when Bol integration support
// was added, so a channel with Type=ChannelTypeBol -- already valid at the
// Go domain level (ChannelType.Valid()) and at the OpenAPI request layer
// (both fixed in earlier commits) -- still failed at INSERT with a raw
// constraint violation. This can only be caught against a real Postgres,
// not sqlmock, which never enforces CHECK constraints.
func TestSalesChannelRepo_Integration_CreateAcceptsBolType(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	var created channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "BOL", Type: channelsDomain.ChannelTypeBol,
			EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})

	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *channelsDomain.SalesChannel
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, channelsDomain.ChannelTypeBol, found.Type)
}

// TestSalesChannelRepo_Integration_PriceAdjustmentRoundTripsAndEnforcesCheckConstraint
// proves the price_adjustment_basis_points column round-trips through real
// Postgres and that its CHECK constraint (migration 148) actually rejects
// an out-of-range value at the database level, not just in application
// code -- sqlmock can't catch either.
func TestSalesChannelRepo_Integration_PriceAdjustmentRoundTripsAndEnforcesCheckConstraint(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	discount := 2000
	var created channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, PriceAdjustmentBasisPoints: &discount,
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)

	var found *channelsDomain.SalesChannel
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.PriceAdjustmentBasisPoints)
	require.Equal(t, 2000, *found.PriceAdjustmentBasisPoints)

	tooHigh := 10001
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		invalid := channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "Invalid", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, PriceAdjustmentBasisPoints: &tooHigh,
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &invalid)
	})
	require.Error(t, err, "the database CHECK constraint must reject a value outside 0-10000")
}

// TestSalesChannelRepo_Integration_QuantityBreaksRoundTripThroughRealJSONB
// proves the quantity_breaks JSONB column (migration 151) round-trips
// through real Postgres on both Create and a wholesale-replace Update --
// sqlmock can't exercise the driver.Valuer/sql.Scanner pair at all.
func TestSalesChannelRepo_Integration_QuantityBreaksRoundTripThroughRealJSONB(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	var created channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll,
			QuantityBreaks: []channelsDomain.QuantityBreak{
				{MinQuantity: 10, DiscountBasisPoints: 1000},
				{MinQuantity: 50, DiscountBasisPoints: 2500},
			},
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)

	var found *channelsDomain.SalesChannel
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, found.QuantityBreaks, 2)
	assert.Equal(t, int64(10), found.QuantityBreaks[0].MinQuantity)
	assert.Equal(t, 2500, found.QuantityBreaks[1].DiscountBasisPoints)

	// Update replaces wholesale -- down to one break, not merged with the
	// two above.
	found.QuantityBreaks = []channelsDomain.QuantityBreak{{MinQuantity: 20, DiscountBasisPoints: 1800}}
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, found.QuantityBreaks, 1)
	assert.Equal(t, int64(20), found.QuantityBreaks[0].MinQuantity)
}

// TestSalesChannelRepo_Integration_StorefrontModeRoundTripsAndDefaultsToHeadless
// proves storefront_mode (migration 187) persists on create and update, and
// that pre-existing rows -- created before the column existed -- read back
// as headless via the column default.
func TestSalesChannelRepo_Integration_StorefrontModeRoundTripsAndDefaultsToHeadless(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	var themed, legacy channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		themed = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "Shop", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, StorefrontMode: channelsDomain.StorefrontModeTheme,
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Create(ctx, &themed); err != nil {
			return err
		}
		tx, _ := txctx.FromContext(ctx)
		return tx.Exec(`INSERT INTO sales_channels (tenant_id, name, type, enrollment_mode) VALUES (?, 'Legacy', 'internal', 'all')`,
			tenant.IDFromContext(ctx)).Error
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByID(ctx, themed.ID)
		if err != nil {
			return err
		}
		assert.Equal(t, channelsDomain.StorefrontModeTheme, found.StorefrontMode)

		found.StorefrontMode = channelsDomain.StorefrontModeHeadless
		if err := repo.Update(ctx, found); err != nil {
			return err
		}
		again, err := repo.FindByID(ctx, themed.ID)
		if err != nil {
			return err
		}
		assert.Equal(t, channelsDomain.StorefrontModeHeadless, again.StorefrontMode)

		all, err := repo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, c := range all {
			if c.Name == "Legacy" {
				legacy = c
			}
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontModeHeadless, legacy.StorefrontMode)
}

func TestSalesChannelRepo_Integration_SignInURLRoundTripsAndIsInternalOnly(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)
	page := "https://shop.example.com/verify"

	var ch channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		ch = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "Headless", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, SignInURL: &page, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &ch)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByID(ctx, ch.ID)
		if err != nil {
			return err
		}
		require.NotNil(t, found.SignInURL)
		assert.Equal(t, page, *found.SignInURL)

		found.SignInURL = nil
		if err := repo.Update(ctx, found); err != nil {
			return err
		}
		again, err := repo.FindByID(ctx, ch.ID)
		if err != nil {
			return err
		}
		assert.Nil(t, again.SignInURL)
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		return tx.Exec(`INSERT INTO sales_channels (tenant_id, name, type, enrollment_mode, sign_in_url) VALUES (?, 'Woo', 'woocommerce', 'all', 'https://x.example')`,
			tenant.IDFromContext(ctx)).Error
	})
	assert.ErrorContains(t, err, "sales_channels_sign_in_url_requires_internal_check")
}

// ” means "the tenant's default currency". The column used to be CHAR(3),
// which Postgres pads: ” came back as "   ", looked like a currency, and
// every product listing on the channel failed its price lookup.
func TestSalesChannelRepo_Integration_BlankCurrencyReadsBackBlank(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	var ch channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		ch = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "No currency", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &ch)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByID(ctx, ch.ID)
		if err != nil {
			return err
		}
		assert.Equal(t, "", found.CurrencyCode)
		return nil
	})
	require.NoError(t, err)
}

func TestSalesChannelRepo_Integration_StorefrontLocaleRoundTripsAndIsChecked(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	var ch channelsDomain.SalesChannel
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		ch = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "Loja", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, StorefrontLocale: channelsDomain.StorefrontLocalePortuguese,
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &ch)
	}))
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByID(ctx, ch.ID)
		require.NoError(t, err)
		assert.Equal(t, channelsDomain.StorefrontLocalePortuguese, found.StorefrontLocale)
		found.StorefrontLocale = channelsDomain.StorefrontLocaleDutch
		require.NoError(t, repo.Update(ctx, found))
		again, err := repo.FindByID(ctx, ch.ID)
		require.NoError(t, err)
		assert.Equal(t, channelsDomain.StorefrontLocaleDutch, again.StorefrontLocale)
		return nil
	}))
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tx, _ := txctx.FromContext(ctx)
		return tx.Exec(`UPDATE sales_channels SET storefront_locale = 'de' WHERE id = ?`, ch.ID).Error
	})
	assert.ErrorContains(t, err, "sales_channels_storefront_locale_check")
}
