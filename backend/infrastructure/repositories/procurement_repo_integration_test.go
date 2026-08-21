//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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

// TestPurchaseOrderLineItemRepo_Integration_ListPurchaseOrderIDsReceivedBySKU
// covers the cost-price rollup gap: a PO marked received/partially_received
// with no inventory.Batch behind it (e.g. via the desktop Edit Purchase
// Order form's status field -- see application/pms's
// RecalculateCostPricesUseCase and application/procurement's
// ListReceivedCostLinesForSKUUseCase) must still surface for cost-basis
// purposes, but only received/partially_received POs, and only the
// product-type line matching the SKU.
func TestPurchaseOrderLineItemRepo_Integration_ListPurchaseOrderIDsReceivedBySKU(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	// A second, distinct SKU for the "wrong SKU, excluded" case --
	// seedProductSKUWarehouse always uses the same fixed SKUCode, so it
	// can't be called twice in one tenant (unique tenant_id+sku_code).
	var otherSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Other Product", Slug: "other-product",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ITEST-SKU-OTHER",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		otherSKUID = sku.ID
		return nil
	})
	require.NoError(t, err)

	makePO := func(ctx context.Context, status procurementDomain.PurchaseOrderStatus, lineSKUID *string, lineType procurementDomain.LineItemType) string {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, supplierRepo.Create(ctx, supplier))
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: status, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, poRepo.Create(ctx, &po))
		li := procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, SKUID: lineSKUID,
			RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500, LineType: lineType,
			CreatedAt: now, UpdatedAt: now,
		}
		require.NoError(t, lineItemRepo.Create(ctx, &li))
		return po.ID
	}

	var receivedPOID, partiallyReceivedPOID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		receivedPOID = makePO(ctx, procurementDomain.PurchaseOrderStatusReceived, &skuID, procurementDomain.LineItemTypeProduct)
		partiallyReceivedPOID = makePO(ctx, procurementDomain.PurchaseOrderStatusPartiallyReceived, &skuID, procurementDomain.LineItemTypeProduct)
		makePO(ctx, procurementDomain.PurchaseOrderStatusDraft, &skuID, procurementDomain.LineItemTypeProduct)         // wrong status, excluded
		makePO(ctx, procurementDomain.PurchaseOrderStatusReceived, &otherSKUID, procurementDomain.LineItemTypeProduct) // wrong SKU, excluded
		makePO(ctx, procurementDomain.PurchaseOrderStatusReceived, &skuID, procurementDomain.LineItemTypeShipping)     // not a product line, excluded
		return nil
	})
	require.NoError(t, err)

	var ids []string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		ids, err = lineItemRepo.ListPurchaseOrderIDsReceivedBySKU(ctx, skuID)
		return err
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{receivedPOID, partiallyReceivedPOID}, ids)
}

// TestPurchaseOrderLineItemRepo_Integration_ListOrderSurvivesUpdateWithTiedCreatedAt
// reproduces the reported bug: a document-parsing batch stamps every line
// item with the exact same CreatedAt (see RunDocumentParsingJobUseCase), so
// sorting by created_at alone leaves ties for Postgres to break by physical
// row order -- which an UPDATE (e.g. linking a SKU) can silently change by
// relocating the updated row's tuple. Position must keep the order stable
// across that update.
func TestPurchaseOrderLineItemRepo_Integration_ListOrderSurvivesUpdateWithTiedCreatedAt(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var poID string
	var lineItems []procurementDomain.PurchaseOrderLineItem
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

		// Same CreatedAt for every row -- exactly what the parse-time batch
		// insert does.
		for i, desc := range []string{"Widget A", "Widget B", "Widget C"} {
			li := procurementDomain.PurchaseOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID, Position: int64(i),
				RawDescription: desc, Quantity: 1, CreatedAt: now, UpdatedAt: now,
			}
			if err := lineItemRepo.Create(ctx, &li); err != nil {
				return err
			}
			lineItems = append(lineItems, li)
		}
		return nil
	})
	require.NoError(t, err)

	var before []procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		before, err = lineItemRepo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, before, 3)
	assert.Equal(t, []string{"Widget A", "Widget B", "Widget C"}, []string{before[0].RawDescription, before[1].RawDescription, before[2].RawDescription})

	// Link the middle item to a SKU -- same write LinkLineItemToSKUUseCase
	// does, the trigger for the reported reordering.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		middle := lineItems[1]
		middle.SKUID = &skuID
		middle.UpdatedAt = time.Now()
		return lineItemRepo.Update(ctx, &middle)
	})
	require.NoError(t, err)

	var after []procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		after, err = lineItemRepo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, after, 3)
	assert.Equal(t, []string{"Widget A", "Widget B", "Widget C"}, []string{after[0].RawDescription, after[1].RawDescription, after[2].RawDescription},
		"order must survive an update to one of the tied-timestamp rows")
}

func TestPurchaseOrderLineItemRepo_Integration_LineTypeRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

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
		lineItem = procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
			RawDescription: "Shipping", Quantity: 1, UnitCostMinorUnits: 1500,
			LineType: procurementDomain.LineItemTypeShipping, LineTypeConfirmed: false,
			CreatedAt: now, UpdatedAt: now,
		}
		return lineItemRepo.Create(ctx, &lineItem)
	})
	require.NoError(t, err)

	var found *procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found.SKUID, "a charge row has no SKU")
	require.Equal(t, procurementDomain.LineItemTypeShipping, found.LineType)
	require.False(t, found.LineTypeConfirmed)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.LineTypeConfirmed = true
		found.UpdatedAt = time.Now()
		return lineItemRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.True(t, found.LineTypeConfirmed)
}

// TestPurchaseOrderLineItemRepo_Integration_SuggestedBatchRoundTrips proves
// a document-parsed lot number/expiry suggestion survives Create+FindByID --
// same informational, human-confirms-later shape as LineType (see
// TestPurchaseOrderLineItemRepo_Integration_LineTypeRoundTrips), never
// written to a real inventory.Batch by this repo layer.
func TestPurchaseOrderLineItemRepo_Integration_SuggestedBatchRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	expiry := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	var lineItem procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		lineItem = procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
			RawDescription: "Farofa Premium", Quantity: 36, UnitCostMinorUnits: 179,
			SuggestedLotNumber: "PT26004", SuggestedExpiryDate: &expiry,
			CreatedAt: now, UpdatedAt: now,
		}
		return lineItemRepo.Create(ctx, &lineItem)
	})
	require.NoError(t, err)

	var found *procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "PT26004", found.SuggestedLotNumber)
	require.NotNil(t, found.SuggestedExpiryDate)
	require.True(t, expiry.Equal(*found.SuggestedExpiryDate))
}

// TestPurchaseOrderLineItemRepo_Integration_SumByPurchaseOrders is the "PO
// total" feature's aggregate query: quantity*unit_cost_minor_units summed
// per PO, one query for many POs at once -- including a discount row (a
// non-negative unit cost, per AddLineItemRequest's openapi minimum:0 --
// SumByPurchaseOrders itself must flip the sign for line_type=discount, see
// its doc comment) and a PO with no line items at all (must come back 0,
// not be missing from the caller's perspective, per SumByPurchaseOrders'
// doc comment -- ListPurchaseOrdersUseCase defaults a missing map entry to
// 0, so either representation works for it, but this still pins the actual
// returned shape).
func TestPurchaseOrderLineItemRepo_Integration_SumByPurchaseOrders(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var poWithItemsID, emptyPOID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}

		poWithItems := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &poWithItems); err != nil {
			return err
		}
		poWithItemsID = poWithItems.ID

		emptyPO := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &emptyPO); err != nil {
			return err
		}
		emptyPOID = emptyPO.ID

		product := procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poWithItemsID, SKUID: &skuID,
			RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500, CreatedAt: now, UpdatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &product); err != nil {
			return err
		}
		discount := procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poWithItemsID,
			RawDescription: "Discount", Quantity: 1, UnitCostMinorUnits: 1000,
			LineType: procurementDomain.LineItemTypeDiscount, CreatedAt: now, UpdatedAt: now,
		}
		return lineItemRepo.Create(ctx, &discount)
	})
	require.NoError(t, err)

	var totals map[string]int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		totals, err = lineItemRepo.SumByPurchaseOrders(ctx, []string{poWithItemsID, emptyPOID})
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(4000), totals[poWithItemsID], "10*500 - 1*1000 (discount subtracts even though its own unit cost is stored positive)")
	_, hasEmpty := totals[emptyPOID]
	assert.False(t, hasEmpty, "a PO with no line items has no row in the aggregate result")
}

func TestPurchaseOrderLineItemRepo_Integration_SKUAutoLinkedFromAliasRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

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
		lineItem = procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, SKUID: &skuID,
			RawDescription: "Widget", Quantity: 10, UnitCostMinorUnits: 500,
			SKUAutoLinkedFromAlias: true,
			CreatedAt:              now, UpdatedAt: now,
		}
		return lineItemRepo.Create(ctx, &lineItem)
	})
	require.NoError(t, err)

	var found *procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.True(t, found.SKUAutoLinkedFromAlias)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.SKUAutoLinkedFromAlias = false
		found.UpdatedAt = time.Now()
		return lineItemRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.False(t, found.SKUAutoLinkedFromAlias, "an explicit human link must clear the auto-linked flag")
}

// TestPurchaseOrderLineItemRepo_Integration_UnitsPerPurchasedItemRoundTrips
// is the "box of a product" edge case (migration 044): a line item's
// multiplier (a supplier's box maps to N stock units of the linked SKU)
// round-trips through Create/FindByID/Update against real Postgres.
func TestPurchaseOrderLineItemRepo_Integration_UnitsPerPurchasedItemRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

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
		lineItem = procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, SKUID: &skuID,
			RawDescription: "Rapadurinha 36x25gr", Quantity: 1, UnitsPerPurchasedItem: 36, UnitCostMinorUnits: 3600,
			CreatedAt: now, UpdatedAt: now,
		}
		return lineItemRepo.Create(ctx, &lineItem)
	})
	require.NoError(t, err)

	var found *procurementDomain.PurchaseOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(36), found.UnitsPerPurchasedItem)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.UnitsPerPurchasedItem = 12
		found.UpdatedAt = time.Now()
		return lineItemRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = lineItemRepo.FindByID(ctx, lineItem.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(12), found.UnitsPerPurchasedItem)
}

// TestSupplierProductAliasRepo_Integration_UnitsPerPurchasedItemRoundTrips
// is the learned side of the "box of a product" edge case: Upsert writes
// the multiplier alongside the SKU link, and a second Upsert for the same
// (tenant, supplier, raw_key) updates it rather than leaving the old value.
func TestSupplierProductAliasRepo_Integration_UnitsPerPurchasedItemRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	supplierRepo := repositories.NewSupplierRepo(db)
	aliasRepo := repositories.NewSupplierProductAliasRepo(db)
	skuID, _ := seedProductSKUWarehouse(t, db, ctx)

	var supplierID string
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
		alias := procurementDomain.SupplierProductAlias{
			TenantID: tenant.IDFromContext(ctx), SupplierID: supplierID, RawKey: "rapadurinha 36x25gr",
			SKUID: skuID, UnitsPerPurchasedItem: 36, CreatedAt: now, UpdatedAt: now,
		}
		return aliasRepo.Upsert(ctx, &alias)
	})
	require.NoError(t, err)

	var found *procurementDomain.SupplierProductAlias
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, "rapadurinha 36x25gr")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(36), found.UnitsPerPurchasedItem)

	// A second link with a corrected multiplier must update in place, not
	// create a duplicate or leave the stale value.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		alias := procurementDomain.SupplierProductAlias{
			TenantID: tenant.IDFromContext(ctx), SupplierID: supplierID, RawKey: "rapadurinha 36x25gr",
			SKUID: skuID, UnitsPerPurchasedItem: 12, CreatedAt: now, UpdatedAt: now,
		}
		return aliasRepo.Upsert(ctx, &alias)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, "rapadurinha 36x25gr")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(12), found.UnitsPerPurchasedItem)
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
