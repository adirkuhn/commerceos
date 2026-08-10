//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedListing creates the connection -> product_channel_listing chain a
// sync log entry's FK needs, reusing the real repos. Returns the listing id
// plus the connection id, since SyncLogEntry needs both as UUID FKs.
func seedListing(t *testing.T, db *gorm.DB, ctx context.Context, productID string) (listingID, connectionID string) {
	t.Helper()
	connRepo := repositories.NewConnectionRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://itest.example.com", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}
		connectionID = conn.ID

		listing := channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: productID, ConnectionID: conn.ID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "999",
			SyncStatus: channelsDomain.SyncStatusSynced, CreatedAt: now, UpdatedAt: now,
		}
		if err := listingRepo.Create(ctx, &listing); err != nil {
			return err
		}
		listingID = listing.ID
		return nil
	})
	require.NoError(t, err)
	return listingID, connectionID
}

func TestSyncLogRepo_Integration_CreateAndListByListing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	listingID, connectionID := seedListing(t, db, ctx, productID)
	syncLogRepo := repositories.NewSyncLogRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		older := channelsDomain.SyncLogEntry{
			TenantID: tenant.IDFromContext(ctx), ListingID: listingID, ProductID: productID, ConnectionID: connectionID, Provider: "woocommerce",
			Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusSuccess,
			Fields: map[string]string{"name": "old"}, Changes: map[string]channelsDomain.FieldChange{},
			SyncedAt: time.Now().Add(-time.Hour),
		}
		if err := syncLogRepo.Create(ctx, &older); err != nil {
			return err
		}
		newer := channelsDomain.SyncLogEntry{
			TenantID: tenant.IDFromContext(ctx), ListingID: listingID, ProductID: productID, ConnectionID: connectionID, Provider: "woocommerce",
			Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusSuccess,
			Fields: map[string]string{"name": "new"}, Changes: map[string]channelsDomain.FieldChange{"name": {From: "old", To: "new"}},
			SyncedAt: time.Now(),
		}
		return syncLogRepo.Create(ctx, &newer)
	})
	require.NoError(t, err)

	var entries []channelsDomain.SyncLogEntry
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		entries, err = syncLogRepo.ListByListing(ctx, listingID, 10)
		return err
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "new", entries[0].Fields["name"], "newest first")
	require.Equal(t, "new", entries[0].Changes["name"].To)
}

func TestSyncLogRepo_Integration_ListAllPaginatesAndCountsAcrossTenant(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	listingID, connectionID := seedListing(t, db, ctx, productID)
	syncLogRepo := repositories.NewSyncLogRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i := 0; i < 3; i++ {
			e := channelsDomain.SyncLogEntry{
				TenantID: tenant.IDFromContext(ctx), ListingID: listingID, ProductID: productID, ConnectionID: connectionID, Provider: "woocommerce",
				Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusSuccess,
				Fields: map[string]string{}, Changes: map[string]channelsDomain.FieldChange{},
				SyncedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			if err := syncLogRepo.Create(ctx, &e); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var page []channelsDomain.SyncLogEntry
	var total int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		page, total, err = syncLogRepo.ListAll(ctx, 1, 2)
		return err
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, page, 2)
}

// skuIDToProductID resolves a SKU back to its product id via the real
// repos (variant lookup), since seedProductSKUWarehouse only returns the
// SKU id.
func skuIDToProductID(t *testing.T, db *gorm.DB, ctx context.Context, skuID string) string {
	t.Helper()
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	var productID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sku, err := skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		variant, err := variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		productID = variant.ProductID
		return nil
	})
	require.NoError(t, err)
	return productID
}

// skuIDToVariantID is skuIDToProductID's sibling for tests that need the
// variant id itself (e.g. to set ProductChannelListing.VariantID or to
// delete the variant and prove the FK's ON DELETE CASCADE).
func skuIDToVariantID(t *testing.T, db *gorm.DB, ctx context.Context, skuID string) string {
	t.Helper()
	skuRepo := repositories.NewSKURepo(db)
	var variantID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sku, err := skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		variantID = sku.VariantID
		return nil
	})
	require.NoError(t, err)
	return variantID
}

// TestProductChannelListingRepo_Integration_VariantRowsRoundTrip proves
// migration 046's variant_id/is_variation columns exist and round-trip
// through the real repo -- both the nil-VariantID "parent" lookup and the
// specific-Variant lookup FindByProductVariantAndConnection promises.
func TestProductChannelListingRepo_Integration_VariantRowsRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	variantID := skuIDToVariantID(t, db, ctx, skuID)
	connRepo := repositories.NewConnectionRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)

	var connectionID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://itest.example.com", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}
		connectionID = conn.ID

		parent := channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: productID, ConnectionID: conn.ID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900",
			SyncStatus: channelsDomain.SyncStatusSynced, CreatedAt: now, UpdatedAt: now,
		}
		if err := listingRepo.Create(ctx, &parent); err != nil {
			return err
		}

		variantRow := channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: productID, ConnectionID: conn.ID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901",
			VariantID: &variantID, IsVariation: true,
			SyncStatus: channelsDomain.SyncStatusSynced, CreatedAt: now, UpdatedAt: now,
		}
		return listingRepo.Create(ctx, &variantRow)
	})
	require.NoError(t, err)

	var parentFound, variantFound *channelsDomain.ProductChannelListing
	var all []channelsDomain.ProductChannelListing
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		parentFound, err = listingRepo.FindByProductVariantAndConnection(ctx, productID, nil, connectionID)
		if err != nil {
			return err
		}
		variantFound, err = listingRepo.FindByProductVariantAndConnection(ctx, productID, &variantID, connectionID)
		if err != nil {
			return err
		}
		all, err = listingRepo.ListByProductAndConnection(ctx, productID, connectionID)
		return err
	})
	require.NoError(t, err)

	require.Equal(t, "900", parentFound.ExternalID)
	require.Nil(t, parentFound.VariantID)
	require.False(t, parentFound.IsVariation)

	require.Equal(t, "901", variantFound.ExternalID)
	require.NotNil(t, variantFound.VariantID)
	require.Equal(t, variantID, *variantFound.VariantID)
	require.True(t, variantFound.IsVariation)

	require.Len(t, all, 2)
}

// TestProductChannelListingRepo_Integration_VariantDeleteCascadesListingRow
// proves migration 046's variant_id FK is ON DELETE CASCADE: deleting the
// Variant a per-variant listing row points at removes that row too,
// without touching the sibling product-level (parent) row.
func TestProductChannelListingRepo_Integration_VariantDeleteCascadesListingRow(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	variantID := skuIDToVariantID(t, db, ctx, skuID)
	connRepo := repositories.NewConnectionRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://itest.example.com", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}

		parent := channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: productID, ConnectionID: conn.ID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900",
			SyncStatus: channelsDomain.SyncStatusSynced, CreatedAt: now, UpdatedAt: now,
		}
		if err := listingRepo.Create(ctx, &parent); err != nil {
			return err
		}

		variantRow := channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: productID, ConnectionID: conn.ID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901",
			VariantID: &variantID, IsVariation: true,
			SyncStatus: channelsDomain.SyncStatusSynced, CreatedAt: now, UpdatedAt: now,
		}
		if err := listingRepo.Create(ctx, &variantRow); err != nil {
			return err
		}

		return variantRepo.Delete(ctx, variantID)
	})
	require.NoError(t, err)

	var remaining []channelsDomain.ProductChannelListing
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		remaining, err = listingRepo.ListByProduct(ctx, productID)
		return err
	})
	require.NoError(t, err)

	require.Len(t, remaining, 1, "the per-variant row must be gone, cascaded from the variant delete")
	require.Equal(t, "900", remaining[0].ExternalID)
	require.Nil(t, remaining[0].VariantID, "the parent row must survive untouched")
}
