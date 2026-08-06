//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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

func TestDocumentParsingJobRepo_Integration_MarkOrphanedRunningAsFailed(t *testing.T) {
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

	adminCtx := tenant.WithSuperAdmin(context.Background())

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, time.Now())
	}))

	var recovered int64
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		recovered, err = jobRepo.MarkOrphanedRunningAsFailed(ctx, "scheduler restarted mid-job", time.Now())
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), recovered)

	var found *procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = jobRepo.FindByID(ctx, job.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.DocumentParsingJobFailed, found.Status)
	require.Equal(t, "scheduler restarted mid-job", found.ErrorMessage)
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
