//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestListing_Integration_CanBeMarkedMissing: migration 244 allows the
// 'missing' sync status (COM-18).
func TestListing_Integration_CanBeMarkedMissing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	_, connectionID := seedListing(t, db, ctx, productID)
	repo := repositories.NewProductChannelListingRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		l, err := repo.FindByProductAndConnection(ctx, productID, connectionID)
		if err != nil {
			return err
		}
		l.SyncStatus = channelsDomain.SyncStatusMissing
		l.LastSyncError = "the product no longer exists in WooCommerce"
		return repo.Update(ctx, l)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		l, err := repo.FindByProductAndConnection(ctx, productID, connectionID)
		if err != nil {
			return err
		}
		assert.Equal(t, channelsDomain.SyncStatusMissing, l.SyncStatus)
		return nil
	})
	require.NoError(t, err)
}
