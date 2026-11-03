//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type planningFixture struct {
	db          *gorm.DB
	ctx         context.Context
	repo        *repositories.PurchasePlanningRepo
	warehouseID string
	now         time.Time
}

func newPlanningFixture(t *testing.T) planningFixture {
	t.Helper()
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	return planningFixture{db: db, ctx: ctx, repo: repositories.NewPurchasePlanningRepo(db), warehouseID: warehouseID, now: time.Now().UTC()}
}

func (f planningFixture) inTx(t *testing.T, fn func(ctx context.Context) error) {
	t.Helper()
	require.NoError(t, tenant.WithTenant(f.ctx, f.db, fn))
}

// sku creates a product with one variant + SKU; mutate adjusts the SKU.
func (f planningFixture) sku(t *testing.T, code string, productStatus pmsDomain.ProductStatus, mutate func(*pmsDomain.SKU)) string {
	t.Helper()
	var id string
	f.inTx(t, func(ctx context.Context) error {
		p := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "P " + code, Slug: "p-" + code, Status: productStatus, CreatedAt: f.now, UpdatedAt: f.now}
		if err := repositories.NewProductRepo(f.db).Create(ctx, p); err != nil {
			return err
		}
		v := &pmsDomain.Variant{TenantID: p.TenantID, ProductID: p.ID, Name: "Default", CreatedAt: f.now, UpdatedAt: f.now}
		if err := repositories.NewVariantRepo(f.db).Create(ctx, v); err != nil {
			return err
		}
		s := &pmsDomain.SKU{TenantID: p.TenantID, VariantID: v.ID, SKUCode: code, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: f.now, UpdatedAt: f.now}
		if mutate != nil {
			mutate(s)
		}
		if err := repositories.NewSKURepo(f.db).Create(ctx, s); err != nil {
			return err
		}
		id = s.ID
		return nil
	})
	return id
}

func (f planningFixture) stock(t *testing.T, skuID string, deltas ...int64) {
	t.Helper()
	f.inTx(t, func(ctx context.Context) error {
		for _, d := range deltas {
			if err := repositories.NewStockMovementRepo(f.db).Create(ctx, &invDomain.StockMovement{
				TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: f.warehouseID,
				Type: invDomain.MovementTypeAdjustment, QuantityDelta: d, OccurredAt: f.now, CreatedAt: f.now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (f planningFixture) sale(t *testing.T, extID, skuID string, status omsDomain.SalesOrderStatus, at time.Time, qty, refunded int64) {
	t.Helper()
	f.inTx(t, func(ctx context.Context) error {
		o := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), 
			WarehouseID: f.warehouseID, Status: status, CurrencyCode: "EUR", TotalMinorUnits: qty * 100, OccurredAt: at, CreatedAt: at,
		}
		if err := repositories.NewSalesOrderRepo(f.db).Create(ctx, &o); err != nil {
			return err
		}
		li := omsDomain.SalesOrderLineItem{
			TenantID: o.TenantID, SalesOrderID: o.ID, SKUID: &skuID, SKUCode: "X", Name: "X",
			Quantity: qty, PriceMinorUnits: 100, TotalMinorUnits: qty * 100, CreatedAt: at,
		}
		lineRepo := repositories.NewSalesOrderLineItemRepo(f.db)
		if err := lineRepo.Create(ctx, &li); err != nil {
			return err
		}
		if refunded > 0 {
			return lineRepo.Update(ctx, li.ID, map[string]any{"refunded_quantity": refunded, "refunded_amount_minor_units": refunded * 100})
		}
		return nil
	})
}

func (f planningFixture) supplier(t *testing.T, name string, leadTime *int) string {
	t.Helper()
	var id string
	f.inTx(t, func(ctx context.Context) error {
		s := &suppliersDomain.Supplier{TenantID: tenant.IDFromContext(ctx), Name: name, LeadTimeDays: leadTime, Status: suppliersDomain.SupplierStatusActive, CreatedAt: f.now, UpdatedAt: f.now}
		err := repositories.NewSupplierRepo(f.db).Create(ctx, s)
		id = s.ID
		return err
	})
	return id
}

func (f planningFixture) purchaseOrder(t *testing.T, supplierID *string, status procurementDomain.PurchaseOrderStatus, createdAt time.Time, lines ...procurementDomain.PurchaseOrderLineItem) {
	t.Helper()
	f.inTx(t, func(ctx context.Context) error {
		po := &procurementDomain.PurchaseOrder{TenantID: tenant.IDFromContext(ctx), SupplierID: supplierID, Status: status, CurrencyCode: "EUR", CreatedAt: createdAt, UpdatedAt: createdAt}
		if err := repositories.NewPurchaseOrderRepo(f.db).Create(ctx, po); err != nil {
			return err
		}
		for i := range lines {
			lines[i].TenantID, lines[i].PurchaseOrderID, lines[i].CreatedAt, lines[i].UpdatedAt = po.TenantID, po.ID, createdAt, createdAt
			if err := repositories.NewPurchaseOrderLineItemRepo(f.db).Create(ctx, &lines[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func productLine(skuID string, qty, perItem, cost int64) procurementDomain.PurchaseOrderLineItem {
	return procurementDomain.PurchaseOrderLineItem{
		SKUID: &skuID, Quantity: qty, UnitsPerPurchasedItem: perItem, UnitCostMinorUnits: cost,
		LineType: procurementDomain.LineItemTypeProduct,
	}
}

func TestPurchasePlanningRepo_Integration_ListPlanningSKUs(t *testing.T) {
	f := newPlanningFixture(t)
	rice := f.sku(t, "RICE", pmsDomain.ProductStatusActive, nil)
	pool := f.sku(t, "MEAT-POOL", pmsDomain.ProductStatusActive, func(s *pmsDomain.SKU) { s.UnitType = pmsDomain.SKUUnitTypeKilo })
	grams := int64(380)
	pkg := f.sku(t, "MEAT-380", pmsDomain.ProductStatusActive, func(s *pmsDomain.SKU) {
		s.PortionedFromSKUID, s.WeightGrams = &pool, &grams
	})
	archived := f.sku(t, "OLD", pmsDomain.ProductStatusArchived, nil)
	inactive := f.sku(t, "OFF", pmsDomain.ProductStatusActive, func(s *pmsDomain.SKU) { s.Status = pmsDomain.SKUStatusInactive })
	f.sku(t, "GIFT", pmsDomain.ProductStatusActive, func(s *pmsDomain.SKU) { s.IsVirtual = true })
	f.stock(t, rice, 10, -3)
	f.stock(t, pool, 5000)

	var skus []planning.PlanningSKU
	f.inTx(t, func(ctx context.Context) (err error) {
		skus, err = f.repo.ListPlanningSKUs(ctx)
		return err
	})

	byID := map[string]planning.PlanningSKU{}
	for _, s := range skus {
		byID[s.SKUID] = s
	}
	assert.Equal(t, int64(7), byID[rice].OnHand)
	assert.True(t, byID[rice].Active)
	assert.Equal(t, "P RICE", byID[rice].ProductName)
	assert.Equal(t, "RICE", byID[rice].SKUCode)
	assert.True(t, byID[pool].IsWeightPool)
	assert.Equal(t, int64(5000), byID[pool].OnHand)
	require.NotNil(t, byID[pkg].PortionedFromSKUID)
	assert.Equal(t, pool, *byID[pkg].PortionedFromSKUID)
	assert.Equal(t, grams, *byID[pkg].WeightGrams)
	assert.False(t, byID[archived].Active)
	assert.False(t, byID[inactive].Active)
	for _, s := range skus {
		assert.NotEqual(t, "GIFT", s.SKUCode, "virtual SKUs are never bought")
	}
}

func TestPurchasePlanningRepo_Integration_Demand(t *testing.T) {
	f := newPlanningFixture(t)
	sku := f.sku(t, "PAN", pmsDomain.ProductStatusActive, nil)
	first := f.now.AddDate(-1, 0, -3)
	f.sale(t, "old", sku, omsDomain.SalesOrderStatusCompleted, first, 4, 0)
	f.sale(t, "in", sku, omsDomain.SalesOrderStatusCompleted, f.now.AddDate(0, 0, -2), 5, 0)
	f.sale(t, "partial", sku, omsDomain.SalesOrderStatusPartiallyRefunded, f.now.AddDate(0, 0, -1), 3, 1)
	f.sale(t, "refunded", sku, omsDomain.SalesOrderStatusRefunded, f.now.AddDate(0, 0, -1), 9, 0)
	f.sale(t, "cancelled", sku, omsDomain.SalesOrderStatusCancelled, f.now.AddDate(-2, 0, 0), 9, 0)

	var sold map[string]int64
	var firstSale map[string]time.Time
	f.inTx(t, func(ctx context.Context) (err error) {
		if sold, err = f.repo.UnitsSoldPerSKUBetween(ctx, f.now.AddDate(0, 0, -7), f.now); err != nil {
			return err
		}
		firstSale, err = f.repo.FirstSalePerSKU(ctx)
		return err
	})

	assert.Equal(t, map[string]int64{sku: 7}, sold) // 5 + (3-1); refunded order excluded
	assert.WithinDuration(t, first, firstSale[sku], time.Second, "cancelled order isn't a sale")
}

func TestPurchasePlanningRepo_Integration_Supply(t *testing.T) {
	f := newPlanningFixture(t)
	pan := f.sku(t, "PAN", pmsDomain.ProductStatusActive, nil)
	choc := f.sku(t, "CHOC", pmsDomain.ProductStatusActive, nil)
	lead := 21
	bauducco := f.supplier(t, "Bauducco", &lead)
	other := f.supplier(t, "Other", nil)

	f.purchaseOrder(t, &other, procurementDomain.PurchaseOrderStatusReceived, f.now.AddDate(0, -3, 0), productLine(pan, 10, 1, 300))
	f.purchaseOrder(t, &bauducco, procurementDomain.PurchaseOrderStatusSent, f.now.AddDate(0, -1, 0),
		productLine(pan, 2, 6, 450),
		procurementDomain.PurchaseOrderLineItem{SKUID: &pan, Quantity: 1, UnitCostMinorUnits: 999, LineType: procurementDomain.LineItemTypeShipping})
	f.purchaseOrder(t, &other, procurementDomain.PurchaseOrderStatusDraft, f.now.AddDate(0, 0, -1), productLine(choc, 3, 1, 200))
	f.purchaseOrder(t, nil, procurementDomain.PurchaseOrderStatusNeedsReview, f.now, productLine(pan, 1, 6, 0))
	f.purchaseOrder(t, &other, procurementDomain.PurchaseOrderStatusCancelled, f.now, productLine(pan, 50, 1, 1))

	var open map[string]planning.OpenOrderQuantity
	var sourcing map[string]planning.Sourcing
	f.inTx(t, func(ctx context.Context) (err error) {
		if open, err = f.repo.OpenOrderQuantities(ctx); err != nil {
			return err
		}
		sourcing, err = f.repo.LatestSourcing(ctx)
		return err
	})

	assert.Equal(t, planning.OpenOrderQuantity{Sent: 12, Draft: 6}, open[pan])
	assert.Equal(t, planning.OpenOrderQuantity{Draft: 3}, open[choc])

	// Newest non-cancelled PO with a supplier; shipping line ignored.
	require.Contains(t, sourcing, pan)
	assert.Equal(t, bauducco, sourcing[pan].SupplierID)
	assert.Equal(t, "Bauducco", sourcing[pan].SupplierName)
	assert.Equal(t, &lead, sourcing[pan].LeadTimeDays)
	assert.Equal(t, int64(450), sourcing[pan].UnitCostMinorUnits)
	assert.Equal(t, int64(6), sourcing[pan].UnitsPerPurchasedItem)
	assert.Equal(t, "EUR", sourcing[pan].CurrencyCode)
	assert.Equal(t, other, sourcing[choc].SupplierID)
	assert.Nil(t, sourcing[choc].LeadTimeDays)
}
