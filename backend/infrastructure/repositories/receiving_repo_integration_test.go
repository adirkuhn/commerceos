//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestReceivingRecordRepo_Integration_CreateFindListUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	recRepo := repositories.NewReceivingRecordRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var poID string
	var created procurementDomain.ReceivingRecord
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
			Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		poID = po.ID

		created = procurementDomain.ReceivingRecord{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
			InvoiceFileURL: "https://example.com/invoice.pdf", InvoiceFileName: "invoice.pdf", ContentType: "application/pdf",
			Status: procurementDomain.ReceivingRecordStatusPendingReview, CreatedAt: now, UpdatedAt: now,
		}
		return recRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *procurementDomain.ReceivingRecord
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = recRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.ReceivingRecordStatusPendingReview, found.Status)

	var listed []procurementDomain.ReceivingRecord
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = recRepo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.Status = procurementDomain.ReceivingRecordStatusApplied
		found.UpdatedAt = time.Now()
		return recRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = recRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.ReceivingRecordStatusApplied, found.Status)
}

func TestReceivingRecordRepo_Integration_NotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	recRepo := repositories.NewReceivingRecordRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := recRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, procurementDomain.ErrReceivingRecordNotFound)
}

func TestReceivingDiscrepancyRepo_Integration_CreateAndList(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	recRepo := repositories.NewReceivingRecordRepo(db)
	discRepo := repositories.NewReceivingDiscrepancyRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var recordID string
	var lineItemID string
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
			Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		li := procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, SKUID: &skuID,
			RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500, CreatedAt: now, UpdatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &li); err != nil {
			return err
		}
		lineItemID = li.ID

		rec := procurementDomain.ReceivingRecord{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
			InvoiceFileURL: "https://example.com/invoice.pdf", Status: procurementDomain.ReceivingRecordStatusPendingReview,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := recRepo.Create(ctx, &rec); err != nil {
			return err
		}
		recordID = rec.ID

		d1 := procurementDomain.ReceivingDiscrepancy{
			TenantID: tenant.IDFromContext(ctx), ReceivingRecordID: recordID, LineItemID: &lineItemID,
			Type: procurementDomain.DiscrepancyTypeQuantityMismatch, ExpectedQty: 10, ReceivedQty: 8,
			Details: "received 8, expected 10", CreatedAt: now,
		}
		if err := discRepo.Create(ctx, &d1); err != nil {
			return err
		}
		d2 := procurementDomain.ReceivingDiscrepancy{
			TenantID: tenant.IDFromContext(ctx), ReceivingRecordID: recordID, LineItemID: nil,
			Type: procurementDomain.DiscrepancyTypeUnexpectedItem, ExpectedQty: 0, ReceivedQty: 3,
			Details: "unexpected item on invoice", CreatedAt: now,
		}
		return discRepo.Create(ctx, &d2)
	})
	require.NoError(t, err)

	var listed []procurementDomain.ReceivingDiscrepancy
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = discRepo.ListByReceivingRecord(ctx, recordID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 2)
	require.Equal(t, procurementDomain.DiscrepancyTypeQuantityMismatch, listed[0].Type)
	require.NotNil(t, listed[0].LineItemID)
	require.Equal(t, procurementDomain.DiscrepancyStatusOpen, listed[0].Status)
	require.Equal(t, procurementDomain.DiscrepancyTypeUnexpectedItem, listed[1].Type)
	require.Nil(t, listed[1].LineItemID)

	var found *procurementDomain.ReceivingDiscrepancy
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = discRepo.FindByID(ctx, listed[0].ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.DiscrepancyStatusOpen, found.Status)
	require.Nil(t, found.AcknowledgedAt)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		found.Status = procurementDomain.DiscrepancyStatusAcknowledged
		found.AcknowledgedAt = &now
		return discRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = discRepo.FindByID(ctx, listed[0].ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.DiscrepancyStatusAcknowledged, found.Status)
	require.NotNil(t, found.AcknowledgedAt)
}

func TestReceivingDiscrepancyRepo_Integration_FindByIDNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	discRepo := repositories.NewReceivingDiscrepancyRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := discRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, procurementDomain.ErrDiscrepancyNotFound)
}

func TestReceivingParsingJobRepo_Integration_Lifecycle(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	recRepo := repositories.NewReceivingRecordRepo(db)
	jobRepo := repositories.NewReceivingParsingJobRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var job procurementDomain.ReceivingParsingJob
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
			Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		rec := procurementDomain.ReceivingRecord{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
			InvoiceFileURL: "https://example.com/invoice.pdf", Status: procurementDomain.ReceivingRecordStatusPendingReview,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := recRepo.Create(ctx, &rec); err != nil {
			return err
		}
		job = procurementDomain.ReceivingParsingJob{
			TenantID: tenant.IDFromContext(ctx), ReceivingRecordID: rec.ID,
			Status: procurementDomain.ReceivingParsingJobPending, CreatedAt: now,
		}
		return jobRepo.Create(ctx, &job)
	})
	require.NoError(t, err)
	require.NotEmpty(t, job.ID)

	adminCtx := tenant.WithSuperAdmin(context.Background())

	var pending []procurementDomain.ReceivingParsingJob
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, pending, 1)

	require.NoError(t, tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, time.Now())
	}))

	// Below maxRetries -- RecoverOrphanedRunning requeues to pending rather
	// than failing outright (see the dedicated
	// TestDocumentParsingJobRepo_Integration_RecoverOrphanedRunning for
	// both branches of this same repo logic, shared by both job types).
	var requeued, failed int64
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, 3, "scheduler restarted", time.Now())
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), requeued)
	require.Equal(t, int64(0), failed)

	var found *procurementDomain.ReceivingParsingJob
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = jobRepo.FindByID(ctx, job.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.ReceivingParsingJobPending, found.Status)
	require.Equal(t, 1, found.RetryCount)
}
