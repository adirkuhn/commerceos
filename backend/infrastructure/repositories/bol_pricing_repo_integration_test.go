//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestBolProductOverrideRepo_Integration_BundleTiersRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewBolProductOverrideRepo(db)
	connID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderBol)
	productID := uuid.NewString()
	price := int64(2500)

	upsert := func(o channelsDomain.BolProductOverride) {
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			o.TenantID, o.ProductID, o.ConnectionID = tenant.IDFromContext(ctx), productID, connID
			o.ConditionCategory, o.FulfilmentMethod = channelsDomain.BolConditionNew, channelsDomain.BolFulfilmentFBR
			o.CreatedAt, o.UpdatedAt = time.Now(), time.Now()
			return repo.Upsert(ctx, &o)
		}))
	}
	find := func() *channelsDomain.BolProductOverride {
		var found *channelsDomain.BolProductOverride
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = repo.FindByProductAndConnection(ctx, productID, connID)
			return err
		}))
		return found
	}

	upsert(channelsDomain.BolProductOverride{
		BundlePricingEnabled: true,
		BundleTiers:          []channelsDomain.BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: &price}, {Quantity: 4}},
	})
	found := find()
	assert.True(t, found.BundlePricingEnabled)
	require.Len(t, found.BundleTiers, 2)
	assert.Equal(t, 2, found.BundleTiers[0].Quantity)
	require.NotNil(t, found.BundleTiers[0].UnitPriceMinorUnits)
	assert.Equal(t, int64(2500), *found.BundleTiers[0].UnitPriceMinorUnits)
	assert.Nil(t, found.BundleTiers[1].UnitPriceMinorUnits, "automatic tier")

	upsert(channelsDomain.BolProductOverride{})
	found = find()
	assert.False(t, found.BundlePricingEnabled)
	assert.Empty(t, found.BundleTiers)
}

func TestConnectionRepo_Integration_BolShippingCostRoundTripsAndRejectsNegative(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewConnectionRepo(db)
	connID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderBol)

	save := func(cost int64) error {
		return tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			c, err := repo.FindByID(ctx, connID)
			if err != nil {
				return err
			}
			c.BolShippingCostMinorUnits = cost
			return repo.Update(ctx, c)
		})
	}

	require.NoError(t, save(495))
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c, err := repo.FindByID(ctx, connID)
		if err != nil {
			return err
		}
		assert.Equal(t, int64(495), c.BolShippingCostMinorUnits)
		return nil
	}))
	assert.Error(t, save(-1), "the CHECK constraint rejects a negative shipping cost")
}
