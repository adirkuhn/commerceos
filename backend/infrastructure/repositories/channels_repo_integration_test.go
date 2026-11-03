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
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedListing creates the connection -> product_channel_listing chain a
// sync log entry's FK needs, reusing the real repos. Returns the listing id
// plus the connection id, since SyncLogEntry needs both as UUID FKs.
func seedListing(t *testing.T, db *gorm.DB, ctx context.Context, productID string) (listingID, connectionID string) {
	t.Helper()
	return seedListingWithStoreURL(t, db, ctx, productID, "https://itest.example.com")
}

// seedListingWithStoreURL is seedListing with an explicit store URL --
// needed when a test seeds more than one connection for the same tenant,
// since (tenant_id, provider, store_url) is unique.
func seedListingWithStoreURL(t *testing.T, db *gorm.DB, ctx context.Context, productID, storeURL string) (listingID, connectionID string) {
	t.Helper()
	connRepo := repositories.NewConnectionRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: integrationsDomain.StoreURL(storeURL), ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
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

// TestImportJobRepo_Integration_RecoverOrphanedRunning mirrors
// TestDocumentParsingJobRepo_Integration_RecoverOrphanedRunning for
// ImportJobRepo -- same shared repo logic (requeue below maxRetries, fail
// once exhausted), proven separately here since ImportJobRepo has its own
// model/table.
func TestImportJobRepo_Integration_RecoverOrphanedRunning(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connRepo := repositories.NewConnectionRepo(db)
	jobRepo := repositories.NewImportJobRepo(db)

	seedConnection := func(t *testing.T, storeURL string) string {
		t.Helper()
		var connID string
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			conn := &integrationsDomain.Connection{
				TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
				StoreURL: integrationsDomain.StoreURL(storeURL), ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
				Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
			}
			if err := connRepo.Create(ctx, conn); err != nil {
				return err
			}
			connID = conn.ID
			return nil
		})
		require.NoError(t, err)
		return connID
	}

	// import_jobs_one_active_per_connection allows at most one pending/
	// running job per connection -- two independent connections so
	// belowCap (which ends the test still pending) and exhausted (created
	// after it) don't collide on that constraint.
	belowCapConnID := seedConnection(t, "https://itest-import-job-1.example.com")
	exhaustedConnID := seedConnection(t, "https://itest-import-job-2.example.com")

	seedJob := func(t *testing.T, connID string) channelsDomain.ImportJob {
		t.Helper()
		var job channelsDomain.ImportJob
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			job = channelsDomain.ImportJob{
				TenantID: tenant.IDFromContext(ctx), ConnectionID: connID,
				DefaultWarehouseID: "00000000-0000-0000-0000-000000000001", DefaultCurrency: "EUR",
				Status: channelsDomain.ImportJobPending, CreatedAt: time.Now(), IsFirstImport: true,
			}
			return jobRepo.Create(ctx, &job)
		})
		require.NoError(t, err)
		return job
	}

	adminCtx := tenant.WithSuperAdmin(context.Background())

	belowCap := seedJob(t, belowCapConnID)
	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, belowCap.ID, time.Now())
	}))

	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, 3, "scheduler restarted mid-job", time.Now())
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), requeued)
	require.Equal(t, int64(0), failed)

	var foundBelowCap *channelsDomain.ImportJob
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		rows, err := jobRepo.ListPending(ctx)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if r.ID == belowCap.ID {
				foundBelowCap = &r
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, foundBelowCap, "requeued job must show up as pending again")
	require.Equal(t, 1, foundBelowCap.RetryCount)
	require.True(t, foundBelowCap.IsFirstImport, "must survive the requeue, RunManualWooImportUseCase depends on it staying stable across retries")

	const maxRetries = 3
	exhausted := seedJob(t, exhaustedConnID)
	for i := 0; i < maxRetries; i++ {
		require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkRunning(ctx, exhausted.ID, time.Now())
		}))
		require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			_, _, err := jobRepo.RecoverOrphanedRunning(ctx, maxRetries, "prior attempt", time.Now())
			return err
		}))
	}
	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, exhausted.ID, time.Now())
	}))

	requeued, failed = 0, 0
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxRetries, "scheduler restarted mid-job", time.Now())
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), requeued)
	require.Equal(t, int64(1), failed)
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
			TenantID: tenant.IDFromContext(ctx), ListingID: &listingID, ProductID: productID, ConnectionID: connectionID, Provider: "woocommerce",
			Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusSuccess,
			Fields: map[string]string{"name": "old"}, Changes: map[string]channelsDomain.FieldChange{},
			SyncedAt: time.Now().Add(-time.Hour),
		}
		if err := syncLogRepo.Create(ctx, &older); err != nil {
			return err
		}
		newer := channelsDomain.SyncLogEntry{
			TenantID: tenant.IDFromContext(ctx), ListingID: &listingID, ProductID: productID, ConnectionID: connectionID, Provider: "woocommerce",
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
				TenantID: tenant.IDFromContext(ctx), ListingID: &listingID, ProductID: productID, ConnectionID: connectionID, Provider: "woocommerce",
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

// TestSyncLogRepo_Integration_ListNeedingAttentionFindsOnlyLatestErrorAndUndismissed
// proves the DISTINCT-ON-based query against real Postgres: a listing whose
// LATEST entry is success (even after an earlier error) is excluded; a
// listing whose latest entry is error but dismissed is excluded; only a
// listing whose latest entry is a live, undismissed error shows up.
func TestSyncLogRepo_Integration_ListNeedingAttentionFindsOnlyLatestErrorAndUndismissed(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	syncLogRepo := repositories.NewSyncLogRepo(db)

	// listing A: error then success -- latest is success, must NOT appear.
	listingA, connA := seedListingWithStoreURL(t, db, ctx, productID, "https://itest-needing-attn-a.example.com")
	// listing B: error, dismissed -- must NOT appear.
	listingB, connB := seedListingWithStoreURL(t, db, ctx, productID, "https://itest-needing-attn-b.example.com")
	// listing C: error, undismissed -- the only one that must appear.
	listingC, connC := seedListingWithStoreURL(t, db, ctx, productID, "https://itest-needing-attn-c.example.com")

	var dismissID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		mk := func(listingID, connID string, status channelsDomain.SyncLogStatus, at time.Time) channelsDomain.SyncLogEntry {
			return channelsDomain.SyncLogEntry{
				TenantID: tenant.IDFromContext(ctx), ListingID: &listingID, ProductID: productID, ConnectionID: connID,
				Provider: "woocommerce", Direction: channelsDomain.SyncFlowPush, Status: status,
				Fields: map[string]string{}, Changes: map[string]channelsDomain.FieldChange{}, SyncedAt: at,
			}
		}
		now := time.Now()

		eA1 := mk(listingA, connA, channelsDomain.SyncLogStatusError, now.Add(-time.Minute))
		if err := syncLogRepo.Create(ctx, &eA1); err != nil {
			return err
		}
		eA2 := mk(listingA, connA, channelsDomain.SyncLogStatusSuccess, now)
		if err := syncLogRepo.Create(ctx, &eA2); err != nil {
			return err
		}

		eB := mk(listingB, connB, channelsDomain.SyncLogStatusError, now)
		if err := syncLogRepo.Create(ctx, &eB); err != nil {
			return err
		}
		dismissID = eB.ID

		eC := mk(listingC, connC, channelsDomain.SyncLogStatusError, now)
		return syncLogRepo.Create(ctx, &eC)
	})
	require.NoError(t, err)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return syncLogRepo.Dismiss(ctx, dismissID)
	}))

	var rows []channelsDomain.SyncLogEntry
	var total int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		rows, total, err = syncLogRepo.ListNeedingAttention(ctx, 1, 20)
		return err
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, rows, 1)
	require.Equal(t, listingC, *rows[0].ListingID)
	require.Equal(t, channelsDomain.SyncLogStatusError, rows[0].Status)
	require.Nil(t, rows[0].DismissedAt)
}

func TestSyncLogRepo_Integration_FindByIDAndDismiss(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)
	productID := skuIDToProductID(t, db, ctx, skuID)
	listingID, connID := seedListing(t, db, ctx, productID)
	syncLogRepo := repositories.NewSyncLogRepo(db)

	var entryID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		e := channelsDomain.SyncLogEntry{
			TenantID: tenant.IDFromContext(ctx), ListingID: &listingID, ProductID: productID, ConnectionID: connID,
			Provider: "woocommerce", Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusError,
			ErrorMessage: "woocommerce request failed", Fields: map[string]string{}, Changes: map[string]channelsDomain.FieldChange{},
			SyncedAt: time.Now(),
		}
		err := syncLogRepo.Create(ctx, &e)
		entryID = e.ID
		return err
	})
	require.NoError(t, err)

	var found *channelsDomain.SyncLogEntry
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = syncLogRepo.FindByID(ctx, entryID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "woocommerce request failed", found.ErrorMessage)
	require.Nil(t, found.DismissedAt)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return syncLogRepo.Dismiss(ctx, entryID)
	}))

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = syncLogRepo.FindByID(ctx, entryID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.DismissedAt)
}

func TestSyncLogRepo_Integration_FindByIDReturnsTypedNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	syncLogRepo := repositories.NewSyncLogRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := syncLogRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000099")
		return err
	})
	require.ErrorIs(t, err, channelsDomain.ErrSyncLogNotFound)
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

// TestSyncScheduleRepo_Integration_RecordRunResult_OnlyAdvancesLastRunAtWhenTold
// proves the fix for orders permanently going unimported after a page-fetch
// failure: LastRunAt (the orders runner's incremental "modified since"
// cursor -- see cmd/scheduler/dispatch.go) must only move forward on a run
// that actually completed. LastRunStatus/LastRunError/NextRunAt must always
// update regardless, so a failing run still stays visible and still retries
// on schedule.
func TestSyncScheduleRepo_Integration_RecordRunResult_OnlyAdvancesLastRunAtWhenTold(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connRepo := repositories.NewConnectionRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)

	var scheduleID, connectionID string
	firstRunAt := time.Now().Add(-time.Hour)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://itest-sync-schedule.example.com", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}
		connectionID = conn.ID

		schedule := &channelsDomain.SyncSchedule{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: conn.ID,
			EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
			Enabled: true, IntervalMinutes: 15, CreatedAt: now, UpdatedAt: now,
		}
		if err := scheduleRepo.Upsert(ctx, schedule); err != nil {
			return err
		}
		scheduleID = schedule.ID

		// Seed a first successful run's LastRunAt directly -- Upsert itself
		// doesn't set it, only RecordRunResult does.
		return scheduleRepo.RecordRunResult(ctx, scheduleID, firstRunAt, "success", "", now, firstRunAt, true)
	})
	require.NoError(t, err)

	// A failed run (a page-fetch error mid-import) must NOT move LastRunAt.
	failedRunAt := time.Now()
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scheduleRepo.RecordRunResult(ctx, scheduleID, failedRunAt, "error", "woocommerce request failed", time.Now(), failedRunAt, false)
	})
	require.NoError(t, err)

	var afterFailure *channelsDomain.SyncSchedule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		schedules, err := scheduleRepo.ListByConnection(ctx, connectionID)
		if err != nil {
			return err
		}
		for i := range schedules {
			if schedules[i].ID == scheduleID {
				afterFailure = &schedules[i]
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, afterFailure)
	require.WithinDuration(t, firstRunAt, *afterFailure.LastRunAt, time.Second, "a failed run must not advance the cursor")
	require.Equal(t, "error", afterFailure.LastRunStatus)
	require.Equal(t, "woocommerce request failed", afterFailure.LastRunError)

	// A subsequent successful run DOES advance it.
	successRunAt := time.Now()
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scheduleRepo.RecordRunResult(ctx, scheduleID, successRunAt, "success", "", time.Now(), successRunAt, true)
	})
	require.NoError(t, err)

	var afterSuccess *channelsDomain.SyncSchedule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		schedules, err := scheduleRepo.ListByConnection(ctx, connectionID)
		if err != nil {
			return err
		}
		for i := range schedules {
			if schedules[i].ID == scheduleID {
				afterSuccess = &schedules[i]
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, afterSuccess)
	require.WithinDuration(t, successRunAt, *afterSuccess.LastRunAt, time.Second, "a successful run must advance the cursor")
	require.Equal(t, "success", afterSuccess.LastRunStatus)
	require.Empty(t, afterSuccess.LastRunError)
}

// TestSyncScheduleRepo_Integration_RecordRunResult_CursorAtIsIndependentOfRanAt
// proves LastRunAt is written from cursorAt, not ranAt -- the orders
// runner's whole point (see dispatch.go's runner doc comment): the cursor
// needs to be Woo's own date_modified_gmt, not this process's wall-clock
// time, so the two must be able to diverge.
func TestSyncScheduleRepo_Integration_RecordRunResult_CursorAtIsIndependentOfRanAt(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connRepo := repositories.NewConnectionRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)

	var scheduleID, connectionID string
	ranAt := time.Now()
	cursorAt := ranAt.Add(-90 * time.Minute) // Woo's own clock, well behind this process's
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://itest-cursor-vs-ranat.example.com", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}
		connectionID = conn.ID

		schedule := &channelsDomain.SyncSchedule{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: conn.ID,
			EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
			Enabled: true, IntervalMinutes: 15, CreatedAt: now, UpdatedAt: now,
		}
		if err := scheduleRepo.Upsert(ctx, schedule); err != nil {
			return err
		}
		scheduleID = schedule.ID
		return scheduleRepo.RecordRunResult(ctx, scheduleID, ranAt, "success", "", ranAt.Add(15*time.Minute), cursorAt, true)
	})
	require.NoError(t, err)

	var schedule *channelsDomain.SyncSchedule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		schedules, err := scheduleRepo.ListByConnection(ctx, connectionID)
		if err != nil {
			return err
		}
		for i := range schedules {
			if schedules[i].ID == scheduleID {
				schedule = &schedules[i]
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, schedule)
	require.NotNil(t, schedule.LastRunAt)
	require.WithinDuration(t, cursorAt, *schedule.LastRunAt, time.Second, "LastRunAt must come from cursorAt, not ranAt")
	require.WithinDuration(t, ranAt, schedule.UpdatedAt, time.Second, "UpdatedAt must still be wall-clock ranAt")
}

func TestExternalCategoryLinkRepo_Integration_UpsertFindListDelete(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	categoryRepo := repositories.NewCategoryRepo(db)
	linkRepo := repositories.NewExternalCategoryLinkRepo(db)

	var categoryID, otherCategoryID, connectionID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL:    integrationsDomain.StoreURL("https://category-link-itest.example.com"),
			ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := repositories.NewConnectionRepo(db).Create(ctx, conn); err != nil {
			return err
		}
		connectionID = conn.ID

		c := &pmsDomain.Category{TenantID: tenantID, Name: "Beverages", Slug: "beverages-link-test", CreatedAt: now, UpdatedAt: now}
		if err := categoryRepo.Create(ctx, c); err != nil {
			return err
		}
		categoryID = c.ID

		c2 := &pmsDomain.Category{TenantID: tenantID, Name: "Snacks", Slug: "snacks-link-test", CreatedAt: now, UpdatedAt: now}
		if err := categoryRepo.Create(ctx, c2); err != nil {
			return err
		}
		otherCategoryID = c2.ID
		return nil
	})
	require.NoError(t, err)

	var linkID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		l := channelsDomain.ExternalCategoryLink{
			TenantID: tenantID, ConnectionID: connectionID, Provider: string(integrationsDomain.ProviderWooCommerce),
			ExternalID: "20", CategoryID: categoryID, CreatedAt: now, UpdatedAt: now,
		}
		if err := linkRepo.Upsert(ctx, &l); err != nil {
			return err
		}
		linkID = l.ID
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, linkID)

	var found *channelsDomain.ExternalCategoryLink
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = linkRepo.FindByConnectionAndExternalID(ctx, connectionID, "20")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, categoryID, found.CategoryID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = linkRepo.FindByConnectionAndCategory(ctx, connectionID, categoryID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "20", found.ExternalID)

	// Upserting the same (connection, external_id) again repoints the
	// existing row to a different category -- same row ID, not a duplicate.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		l := channelsDomain.ExternalCategoryLink{
			TenantID: tenantID, ConnectionID: connectionID, Provider: string(integrationsDomain.ProviderWooCommerce),
			ExternalID: "20", CategoryID: otherCategoryID, CreatedAt: now, UpdatedAt: now,
		}
		return linkRepo.Upsert(ctx, &l)
	})
	require.NoError(t, err)

	var listed []channelsDomain.ExternalCategoryLink
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = linkRepo.ListByConnection(ctx, connectionID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, linkID, listed[0].ID)
	require.Equal(t, otherCategoryID, listed[0].CategoryID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return linkRepo.Delete(ctx, linkID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := linkRepo.FindByConnectionAndExternalID(ctx, connectionID, "20")
		return err
	})
	require.ErrorIs(t, err, channelsDomain.ErrExternalCategoryLinkNotFound)
}

// TestSyncScheduleRepo_Integration_RecordHealth_RoundTripsTheFailureStreak:
// the failing streak survives a reload and an Upsert (editing the schedule
// must not silently reset it), and clears on a healthy write.
func TestSyncScheduleRepo_Integration_RecordHealth_RoundTripsTheFailureStreak(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connRepo := repositories.NewConnectionRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)

	var schedule *channelsDomain.SyncSchedule
	since := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Microsecond)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://itest-sync-health.example.com", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}
		schedule = &channelsDomain.SyncSchedule{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: conn.ID,
			EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
			Enabled: true, IntervalMinutes: 15, CreatedAt: now, UpdatedAt: now,
		}
		if err := scheduleRepo.Upsert(ctx, schedule); err != nil {
			return err
		}
		if err := scheduleRepo.RecordHealth(ctx, schedule.ID, &since, true); err != nil {
			return err
		}
		schedule.IntervalMinutes = 30
		return scheduleRepo.Upsert(ctx, schedule)
	})
	require.NoError(t, err)

	reload := func() channelsDomain.SyncSchedule {
		var got []channelsDomain.SyncSchedule
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			got, err = scheduleRepo.ListByConnection(ctx, schedule.ConnectionID)
			return err
		}))
		require.Len(t, got, 1)
		return got[0]
	}

	failing := reload()
	require.NotNil(t, failing.FailingSince)
	require.WithinDuration(t, since, *failing.FailingSince, time.Second)
	require.True(t, failing.FailureAlerted)
	require.Equal(t, 30, failing.IntervalMinutes)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scheduleRepo.RecordHealth(ctx, schedule.ID, nil, false)
	}))
	healthy := reload()
	require.Nil(t, healthy.FailingSince)
	require.False(t, healthy.FailureAlerted)
}
