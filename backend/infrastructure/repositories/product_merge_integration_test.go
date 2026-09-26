//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

func newMergeUseCase(db *gorm.DB) *apppms.MergeProductsUseCase {
	skuRepo := repositories.NewSKURepo(db)
	return apppms.NewMergeProductsUseCase(db, apppms.MergeProductsDeps{
		Products: repositories.NewProductRepo(db), Variants: repositories.NewVariantRepo(db), SKUs: skuRepo,
		OnHand: appinventory.NewListBatchesUseCase(db, repositories.NewBatchRepo(db), repositories.NewStockMovementRepo(db), repositories.NewWarehouseRepo(db)),
		SKUReassigners: []apppms.SKUReferenceReassigner{
			repositories.NewInventorySKUReferences(), repositories.NewProcurementSKUReferences(), repositories.NewOrderSKUReferences(),
			repositories.NewPromotionSKUReferences(), repositories.NewCatalogSKUReferences(),
		},
		ProductReassigners: []apppms.ProductReferenceReassigner{repositories.NewCatalogProductReferences()},
		Ledger:             repositories.NewSKUCostPriceLedgerRepo(db),
		CodeAliases:        repositories.NewMergeCodeAliases(repositories.NewBarcodeAliasRepo(db), repositories.NewScaleCodeAliasRepo(db)),
		Records:            repositories.NewProductMergeRepo(),
		Listings:           repositories.NewProductChannelListingRepo(db),
		Outbox:             repositories.NewOutboxRepository(db),
	})
}

// TestMergeProducts_Integration runs a whole merge against real Postgres:
// a duplicate product booked in with its own lot (same lot/expiry as the
// existing SKU's), some un-lotted stock and a sale folds into the existing
// product -- one batch, all stock, the sale, a blended cost, the old
// barcode scanning to the existing SKU, and the duplicate gone.
func TestMergeProducts_Integration(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	targetSKUID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	expiry := time.Date(2027, 11, 26, 0, 0, 0, 0, time.UTC)

	var sourceProductID, targetProductID, sourceSKUID, targetBatchID string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		tid := tenant.IDFromContext(ctx)
		target, err := skuRepo.FindByID(ctx, targetSKUID)
		if err != nil {
			return err
		}
		targetVariant, err := variantRepo.FindByID(ctx, target.VariantID)
		if err != nil {
			return err
		}
		targetProductID = targetVariant.ProductID
		if err := skuRepo.UpdateCostPriceFromReceipt(ctx, targetSKUID, 100); err != nil {
			return err
		}

		product := &pmsDomain.Product{TenantID: tid, Name: "Duplicate", Slug: "duplicate-merge", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		sourceProductID = product.ID
		variant := &pmsDomain.Variant{TenantID: tid, ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tid, VariantID: variant.ID, SKUCode: "DUP-1", Barcode: "8710001000017",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		sourceSKUID = sku.ID
		if err := skuRepo.UpdateCostPriceFromReceipt(ctx, sourceSKUID, 200); err != nil {
			return err
		}

		batch := func(skuID string) (string, error) {
			b := invDomain.Batch{TenantID: tid, SKUID: skuID, WarehouseID: warehouseID, LotNumber: "L26146", ExpiryDate: &expiry,
				ReceivedAt: now, Status: invDomain.BatchStatusActive, CreatedAt: now, UpdatedAt: now}
			if err := batchRepo.Create(ctx, &b); err != nil {
				return "", err
			}
			return b.ID, nil
		}
		move := func(skuID string, batchID *string, qty int64) error {
			return movementRepo.Create(ctx, &invDomain.StockMovement{TenantID: tid, SKUID: skuID, WarehouseID: warehouseID, BatchID: batchID,
				Type: invDomain.MovementTypeReceipt, QuantityDelta: qty, OccurredAt: now, CreatedAt: now})
		}
		if targetBatchID, err = batch(targetSKUID); err != nil {
			return err
		}
		sourceBatchID, err := batch(sourceSKUID)
		if err != nil {
			return err
		}
		if err := move(targetSKUID, &targetBatchID, 6); err != nil {
			return err
		}
		if err := move(sourceSKUID, &sourceBatchID, 4); err != nil {
			return err
		}
		if err := move(sourceSKUID, nil, 2); err != nil {
			return err
		}

		order := omsDomain.SalesOrder{TenantID: tid, Source: omsDomain.SalesOrderSourcePOS, ExternalOrderID: "pos-merge-1",
			WarehouseID: warehouseID, CurrencyCode: "USD", OccurredAt: now, CreatedAt: now}
		if err := repositories.NewSalesOrderRepo(db).Create(ctx, &order); err != nil {
			return err
		}
		return repositories.NewSalesOrderLineItemRepo(db).Create(ctx, &omsDomain.SalesOrderLineItem{
			TenantID: tid, SalesOrderID: order.ID, SKUID: &sourceSKUID, Quantity: 1, PriceMinorUnits: 500, CreatedAt: now})
	}))

	result, err := newMergeUseCase(db).Execute(ctx, apppms.MergeProductsInput{SourceProductID: sourceProductID, TargetProductID: targetProductID, MergedBy: "user-1"})
	require.NoError(t, err)
	assert.Equal(t, targetProductID, result.TargetProductID)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := productRepo.FindByID(ctx, sourceProductID)
		assert.ErrorIs(t, err, pmsDomain.ErrProductNotFound, "the duplicate is gone")

		batches, err := batchRepo.ListBySKU(ctx, targetSKUID)
		require.NoError(t, err)
		require.Len(t, batches, 1, "the duplicate's same-lot batch folded into the existing one")
		assert.Equal(t, targetBatchID, batches[0].ID)

		movements, err := movementRepo.ListBySKU(ctx, targetSKUID)
		require.NoError(t, err)
		var total, inBatch int64
		for _, m := range movements {
			total += m.QuantityDelta
			if m.BatchID != nil && *m.BatchID == targetBatchID {
				inBatch += m.QuantityDelta
			}
		}
		assert.Equal(t, int64(12), total)
		assert.Equal(t, int64(10), inBatch)

		sku, err := skuRepo.FindByID(ctx, targetSKUID)
		require.NoError(t, err)
		require.NotNil(t, sku.CostPriceMinorUnits)
		// (6*100 + 6*200) / 12 = 150
		assert.Equal(t, int64(150), *sku.CostPriceMinorUnits)

		alias, err := repositories.NewBarcodeAliasRepo(db).FindByCode(ctx, "8710001000017")
		require.NoError(t, err)
		assert.Equal(t, targetSKUID, alias.SKUID, "the duplicate's barcode scans to the existing SKU")

		tx, _ := txctx.FromContext(ctx)
		var sold, records int64
		require.NoError(t, tx.Raw(`SELECT count(*) FROM sales_order_line_items WHERE sku_id = ?`, targetSKUID).Scan(&sold).Error)
		assert.Equal(t, int64(1), sold, "the sale now belongs to the existing SKU")
		require.NoError(t, tx.Raw(`SELECT count(*) FROM product_merges WHERE source_product_id = ? AND target_product_id = ?`, sourceProductID, targetProductID).Scan(&records).Error)
		assert.Equal(t, int64(1), records, "the merge is recorded")
		return nil
	}))
}
