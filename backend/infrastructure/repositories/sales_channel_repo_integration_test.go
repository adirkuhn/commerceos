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
