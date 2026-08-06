//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestPurchaseOrderRepo_Integration_CreateFindListUpdateDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	_ = skuID

	var supplierID string
	var created procurementDomain.PurchaseOrder
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

		created = procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplierID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD",
			CreatedAt: now, UpdatedAt: now,
		}
		return poRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.PurchaseOrderStatusDraft, found.Status)

	var listed []procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = poRepo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.Status = procurementDomain.PurchaseOrderStatusSent
		found.UpdatedAt = time.Now()
		return poRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.PurchaseOrderStatusSent, found.Status)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return poRepo.Delete(ctx, created.ID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

// TestPurchaseOrderRepo_Integration_NilSupplierRoundTrips proves migration
// 029 (supplier_id DROP NOT NULL) actually took effect against real
// Postgres -- a PO uploaded without a supplier picked (see
// CreatePurchaseOrderFromDocumentUseCase) must be creatable and readable
// with a nil SupplierID, then settable via the review UI's one-click
// "create supplier" action (CreateSupplierAndAssignToPurchaseOrderUseCase).
func TestPurchaseOrderRepo_Integration_NilSupplierRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var created procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: nil, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusNeedsReview, CurrencyCode: "USD",
			CreatedAt: now, UpdatedAt: now,
		}
		return poRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found.SupplierID)
	require.Equal(t, procurementDomain.PurchaseOrderStatusNeedsReview, found.Status)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Late-Assigned Supplier", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		found.SupplierID = &supplier.ID
		found.UpdatedAt = now
		return poRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.SupplierID)
}

func TestPurchaseOrderRepo_Integration_NilWarehouseRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var created procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		created = procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: nil,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD",
			CreatedAt: now, UpdatedAt: now,
		}
		return poRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found.WarehouseID, "a PO created with no warehouse must round-trip as nil, not an empty-string uuid")

	// Assigning a warehouse at receiving/booking time is a plain Update,
	// same as any other field.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.WarehouseID = &warehouseID
		found.UpdatedAt = time.Now()
		return poRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.WarehouseID)
	require.Equal(t, warehouseID, *found.WarehouseID)
}

func TestPurchaseOrderRepo_Integration_UpdateSuggestionsDoesNotTouchOtherFields(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	poRepo := repositories.NewPurchaseOrderRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var created procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: nil, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusNeedsReview, CurrencyCode: "USD", Notes: "original notes",
			CreatedAt: now, UpdatedAt: now,
		}
		return poRepo.Create(ctx, &created)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return poRepo.UpdateSuggestions(ctx, created.ID, "Pedaços do Brasil, Lda.", "EUR")
	})
	require.NoError(t, err)

	var found *procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = poRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, "Pedaços do Brasil, Lda.", found.SuggestedSupplierName)
	assert.Equal(t, "EUR", found.SuggestedCurrencyCode)
	// UpdateSuggestions must be scoped to only its two columns -- everything
	// else set at Create time must be untouched.
	assert.Equal(t, "USD", found.CurrencyCode)
	assert.Equal(t, "original notes", found.Notes)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusNeedsReview, found.Status)
}

func TestPurchaseOrderLineItemRepo_Integration_CreateListUpdateDeleteAndCascade(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var poID string
	var lineItem procurementDomain.PurchaseOrderLineItem
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

		lineItem = procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID, SKUID: &skuID,
			RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500,
			CreatedAt: now, UpdatedAt: now,
		}
		return lineItemRepo.Create(ctx, &lineItem)
	})
	require.NoError(t, err)
	require.NotEmpty(t, lineItem.ID)

	var listed []procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = lineItemRepo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, int64(10), listed[0].Quantity)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		lineItem.Quantity = 20
		lineItem.UpdatedAt = time.Now()
		return lineItemRepo.Update(ctx, &lineItem)
	})
	require.NoError(t, err)

	var found *procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(20), found.Quantity)

	// Deleting the parent PO must cascade-delete its line items.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return poRepo.Delete(ctx, poID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.ErrorIs(t, err, procurementDomain.ErrLineItemNotFound, "line item must be gone once its PO is deleted (ON DELETE CASCADE)")
}

// TestPurchaseOrderRepo_Integration_RLSHidesOtherTenantsPurchaseOrders is
// routed through commerceos_app (not the migration-owning superuser) -- RLS
// is never enforced against a superuser regardless of policy.
func TestPurchaseOrderRepo_Integration_RLSHidesOtherTenantsPurchaseOrders(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Tenant A Supplier", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		return poRepo.Create(ctx, &po)
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	var listedForB []procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		listedForB, err = poRepo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, listedForB, "RLS must prevent tenant B from seeing tenant A's purchase order")
}
