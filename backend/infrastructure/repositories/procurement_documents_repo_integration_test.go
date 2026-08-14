//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestPurchaseOrderDocumentRepo_Integration_CreateFindListAndCascade(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	docRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var poID string
	var doc procurementDomain.PurchaseOrderDocument
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		poID = po.ID

		doc = procurementDomain.PurchaseOrderDocument{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
			FileURL: "https://example.com/order.csv", FileName: "order.csv", ContentType: "text/csv",
			CreatedAt: now,
		}
		return docRepo.Create(ctx, &doc)
	})
	require.NoError(t, err)
	require.NotEmpty(t, doc.ID)

	var found *procurementDomain.PurchaseOrderDocument
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = docRepo.FindByID(ctx, doc.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "order.csv", found.FileName)

	var listed []procurementDomain.PurchaseOrderDocument
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = docRepo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return poRepo.Delete(ctx, poID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := docRepo.FindByID(ctx, doc.ID)
		return err
	})
	require.ErrorIs(t, err, procurementDomain.ErrDocumentNotFound, "document must be gone once its PO is deleted (ON DELETE CASCADE)")
}

func TestDocumentParsingJobRepo_Integration_Lifecycle(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	docRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	jobRepo := repositories.NewDocumentParsingJobRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var job procurementDomain.DocumentParsingJob
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		doc := procurementDomain.PurchaseOrderDocument{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
			FileURL: "https://example.com/order.csv", FileName: "order.csv", ContentType: "text/csv", CreatedAt: now,
		}
		if err := docRepo.Create(ctx, &doc); err != nil {
			return err
		}
		job = procurementDomain.DocumentParsingJob{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, DocumentID: doc.ID,
			Status: procurementDomain.DocumentParsingJobPending, CreatedAt: now,
		}
		return jobRepo.Create(ctx, &job)
	})
	require.NoError(t, err)
	require.NotEmpty(t, job.ID)

	// ListPending is a cross-tenant/system query -- run under
	// tenant.WithSuperAdmin, same as cmd/scheduler's real usage.
	adminCtx := tenant.WithSuperAdmin(context.Background())

	var pending []procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, job.ID, pending[0].ID)

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, time.Now())
	}))

	var pendingAfterRunning []procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pendingAfterRunning, err = jobRepo.ListPending(ctx)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, pendingAfterRunning, "a running job must not show up in ListPending")

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, time.Now())
	}))

	var found *procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = jobRepo.FindByID(ctx, job.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.DocumentParsingJobCompleted, found.Status)
	require.NotNil(t, found.FinishedAt)
}

// TestDocumentParsingJobRepo_Integration_RecoverOrphanedRunning proves both
// branches against real Postgres: a job below maxRetries is requeued to
// pending (with retry_count incremented and started_at cleared) instead of
// failed outright, so the scheduler's next poll picks it back up
// automatically; a job that has already exhausted its retries is failed,
// so a job whose own execution reliably crashes the scheduler can't
// requeue itself forever.
func TestDocumentParsingJobRepo_Integration_RecoverOrphanedRunning(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	docRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	jobRepo := repositories.NewDocumentParsingJobRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	seedJob := func(t *testing.T) procurementDomain.DocumentParsingJob {
		t.Helper()
		var job procurementDomain.DocumentParsingJob
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			supplier := &suppliersDomain.Supplier{
				TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
				CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, supplierRepo.Create(ctx, supplier))
			po := procurementDomain.PurchaseOrder{
				TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
				Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, poRepo.Create(ctx, &po))
			doc := procurementDomain.PurchaseOrderDocument{
				TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
				FileURL: "https://example.com/order.csv", FileName: "order.csv", ContentType: "text/csv", CreatedAt: now,
			}
			require.NoError(t, docRepo.Create(ctx, &doc))
			job = procurementDomain.DocumentParsingJob{
				TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, DocumentID: doc.ID,
				Status: procurementDomain.DocumentParsingJobPending, CreatedAt: now,
			}
			return jobRepo.Create(ctx, &job)
		})
		require.NoError(t, err)
		return job
	}

	adminCtx := tenant.WithSuperAdmin(context.Background())

	belowCap := seedJob(t)
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

	var foundBelowCap *procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		foundBelowCap, err = jobRepo.FindByID(ctx, belowCap.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.DocumentParsingJobPending, foundBelowCap.Status, "below maxRetries -- must requeue, not fail")
	require.Equal(t, 1, foundBelowCap.RetryCount)
	require.Nil(t, foundBelowCap.StartedAt, "cleared so it reads as not-yet-attempted again")

	const maxRetries = 3
	exhausted := seedJob(t)
	// Simulate maxRetries prior crash-and-requeue cycles having already
	// happened for this job -- retry_count reaches maxRetries, so the next
	// recovery pass must fail it instead of requeuing again.
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

	requeued, failed, err = 0, 0, nil
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, 3, "scheduler restarted mid-job", time.Now())
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), requeued)
	require.Equal(t, int64(1), failed)

	var foundExhausted *procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		foundExhausted, err = jobRepo.FindByID(ctx, exhausted.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.DocumentParsingJobFailed, foundExhausted.Status)
	require.Equal(t, "scheduler restarted mid-job", foundExhausted.ErrorMessage)
}

func TestSupplierProductAliasRepo_Integration_UpsertAndFind(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	aliasRepo := repositories.NewSupplierProductAliasRepo(db)
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)

	var supplierID string
	var alias procurementDomain.SupplierProductAlias
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		supplierID = supplier.ID
		alias = procurementDomain.SupplierProductAlias{
			TenantID: tenant.IDFromContext(ctx), SupplierID: supplierID, RawKey: "acme widget a", SKUID: skuID,
			CreatedAt: now, UpdatedAt: now,
		}
		return aliasRepo.Upsert(ctx, &alias)
	})
	require.NoError(t, err)
	require.NotEmpty(t, alias.ID)

	var found *procurementDomain.SupplierProductAlias
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, "acme widget a")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, skuID, found.SKUID)

	// Not-found returns (nil, nil), not an error -- the matching pipeline
	// treats "no alias yet" as a normal fall-through case, not a failure.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, "unknown key")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)

	// Re-upserting the same (tenant, supplier, raw_key) updates in place
	// rather than erroring, per the unique constraint + ON CONFLICT clause.
	// seedProductSKUWarehouse can't be called twice under the same tenant
	// (fixed slug/sku_code collide), so the second SKU is seeded inline
	// with distinct values.
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	var otherSKUID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Second Product", Slug: "second-product",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{
			TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now,
		}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ITEST-SKU-2",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		otherSKUID = sku.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		updated := procurementDomain.SupplierProductAlias{
			TenantID: tenant.IDFromContext(ctx), SupplierID: supplierID, RawKey: "acme widget a", SKUID: otherSKUID,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		return aliasRepo.Upsert(ctx, &updated)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, "acme widget a")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, otherSKUID, found.SKUID)
}

// TestPurchaseOrderDocumentRepo_Integration_DuplicateDetectionAcrossManyPOs
// reproduces a real report: upload the same file content into 5 separate
// POs (like AttachDocumentToPurchaseOrderUseCase does per PO), and every
// PO after the first must come back flagged -- not just the second one.
// Runs against real Postgres to catch anything a stub-based unit test
// (fixed duplicateCandidates list) can't: the repo's tenant-wide
// FindDuplicateCandidates query has to keep finding every prior match as
// the document set grows, not just the first.
func TestPurchaseOrderDocumentRepo_Integration_DuplicateDetectionAcrossManyPOs(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	poRepo := repositories.NewPurchaseOrderRepo(db)
	docRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	const contentHash = "same-file-hash"

	var poIDs []string
	for i := 0; i < 5; i++ {
		var poID string
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			po := procurementDomain.PurchaseOrder{
				TenantID: tenant.IDFromContext(ctx), Status: procurementDomain.PurchaseOrderStatusNeedsReview,
				CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
			}
			if err := poRepo.Create(ctx, &po); err != nil {
				return err
			}
			poID = po.ID

			// Same shape as AttachDocumentToPurchaseOrderUseCase.Execute:
			// look up duplicates *before* creating this PO's own document
			// row (so it never matches itself), then mark the PO if any
			// match belongs to a different PO.
			duplicates, err := docRepo.FindDuplicateCandidates(ctx, contentHash, poID)
			if err != nil {
				return err
			}
			doc := procurementDomain.PurchaseOrderDocument{
				TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
				FileURL: "https://example.com/order.pdf", FileName: "order.pdf", ContentType: "application/pdf",
				ContentHash: contentHash, IsPotentialDuplicate: len(duplicates) > 0, CreatedAt: now,
			}
			if err := docRepo.Create(ctx, &doc); err != nil {
				return err
			}
			for _, d := range duplicates {
				if d.PurchaseOrderID != poID {
					if err := poRepo.MarkPotentialDuplicate(ctx, poID, d.PurchaseOrderID); err != nil {
						return err
					}
					break
				}
			}
			return nil
		})
		require.NoError(t, err)
		poIDs = append(poIDs, poID)
	}

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i, poID := range poIDs {
			po, err := poRepo.FindByID(ctx, poID)
			if err != nil {
				return err
			}
			if i == 0 {
				assert.False(t, po.IsPotentialDuplicate, "the first upload has nothing to be a duplicate of")
			} else {
				assert.True(t, po.IsPotentialDuplicate, "PO #%d (uploaded after %d earlier POs with the same file) must be flagged", i+1, i)
				assert.NotNil(t, po.DuplicateOfPurchaseOrderID, "PO #%d", i+1)
			}
		}
		return nil
	})
	require.NoError(t, err)
}

// TestDeletePurchaseOrder_Integration_RepointsRemainingDuplicatesAfterDelete
// is the second half of the same report: after deleting the PO another one
// was flagged as a duplicate of, the survivor must not go stuck-flagged
// with a dangling pointer (ON DELETE SET NULL clears the FK but not
// IsPotentialDuplicate on its own) -- DeletePurchaseOrderUseCase has to
// recompute it.
func TestDeletePurchaseOrder_Integration_RepointsRemainingDuplicatesAfterDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	poRepo := repositories.NewPurchaseOrderRepo(db)
	docRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	deleteUC := appprocurement.NewDeletePurchaseOrderUseCase(db, poRepo, docRepo)
	const contentHash = "same-file-hash"

	var poIDs []string
	for i := 0; i < 3; i++ {
		var poID string
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			po := procurementDomain.PurchaseOrder{
				TenantID: tenant.IDFromContext(ctx), Status: procurementDomain.PurchaseOrderStatusNeedsReview,
				CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
			}
			if err := poRepo.Create(ctx, &po); err != nil {
				return err
			}
			poID = po.ID

			duplicates, err := docRepo.FindDuplicateCandidates(ctx, contentHash, poID)
			if err != nil {
				return err
			}
			doc := procurementDomain.PurchaseOrderDocument{
				TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
				FileURL: "https://example.com/order.pdf", FileName: "order.pdf", ContentType: "application/pdf",
				ContentHash: contentHash, IsPotentialDuplicate: len(duplicates) > 0, CreatedAt: now,
			}
			if err := docRepo.Create(ctx, &doc); err != nil {
				return err
			}
			for _, d := range duplicates {
				if d.PurchaseOrderID != poID {
					if err := poRepo.MarkPotentialDuplicate(ctx, poID, d.PurchaseOrderID); err != nil {
						return err
					}
					break
				}
			}
			return nil
		})
		require.NoError(t, err)
		poIDs = append(poIDs, poID)
	}

	// Delete the first PO -- at least one of the other two necessarily
	// points at it (duplicate resolution order against Postgres isn't
	// guaranteed, so which one isn't asserted, only that both survivors
	// stay correctly flagged against each other afterward).
	require.NoError(t, deleteUC.Execute(ctx, poIDs[0]))

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, poID := range poIDs[1:] {
			po, err := poRepo.FindByID(ctx, poID)
			if err != nil {
				return err
			}
			assert.True(t, po.IsPotentialDuplicate, "PO %s must still be flagged -- it still shares content with the other surviving PO", poID)
			if po.DuplicateOfPurchaseOrderID != nil {
				assert.NotEqual(t, poIDs[0], *po.DuplicateOfPurchaseOrderID, "must not point at the now-deleted PO")
			}
		}
		return nil
	})
	require.NoError(t, err)
}
