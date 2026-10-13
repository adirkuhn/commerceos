//go:build integration

package repositories_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// stockFixture seeds products with SKUs and stock movements for the
// low/out-of-stock filter tests.
type stockFixture struct {
	t           *testing.T
	db          *gorm.DB
	ctx         context.Context
	warehouseID string
	products    *repositories.ProductRepo
}

func newStockFixture(t *testing.T, db *gorm.DB, ctx context.Context) *stockFixture {
	t.Helper()
	f := &stockFixture{t: t, db: db, ctx: ctx, products: repositories.NewProductRepo(db)}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		w := &invDomain.Warehouse{TenantID: tenant.IDFromContext(ctx), Name: "Stock filter WH", Active: true, CreatedAt: now, UpdatedAt: now}
		if err := repositories.NewWarehouseRepo(db).Create(ctx, w); err != nil {
			return err
		}
		f.warehouseID = w.ID
		return nil
	}))
	return f
}

// product creates a product whose SKUs hold the given stock totals (one SKU
// per entry; each total can be split over several movements by the caller
// via addStock).
func (f *stockFixture) product(name string, status pmsDomain.ProductStatus, skuTotals ...int64) (productID string, skuIDs []string) {
	f.t.Helper()
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		now := time.Now()
		p := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: name, Slug: name, Status: status, CreatedAt: now, UpdatedAt: now,
		}
		if err := f.products.Create(ctx, p); err != nil {
			return err
		}
		productID = p.ID
		v := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: p.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := repositories.NewVariantRepo(f.db).Create(ctx, v); err != nil {
			return err
		}
		for i, total := range skuTotals {
			sku := &pmsDomain.SKU{
				TenantID: tenant.IDFromContext(ctx), VariantID: v.ID, SKUCode: name + "-" + string(rune('A'+i)),
				CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := repositories.NewSKURepo(f.db).Create(ctx, sku); err != nil {
				return err
			}
			skuIDs = append(skuIDs, sku.ID)
			if total != 0 {
				if err := f.move(ctx, sku.ID, total); err != nil {
					return err
				}
			}
		}
		return nil
	}))
	return productID, skuIDs
}

func (f *stockFixture) addStock(skuID string, delta int64) {
	f.t.Helper()
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error { return f.move(ctx, skuID, delta) }))
}

func (f *stockFixture) move(ctx context.Context, skuID string, delta int64) error {
	now := time.Now()
	return repositories.NewStockMovementRepo(f.db).Create(ctx, &invDomain.StockMovement{
		TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: f.warehouseID,
		Type: invDomain.MovementTypeAdjustment, QuantityDelta: delta, OccurredAt: now, CreatedAt: now,
	})
}

func (f *stockFixture) setOwnThreshold(productID string, threshold *int) {
	f.t.Helper()
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		return f.products.SetLowStockThreshold(ctx, productID, threshold)
	}))
}

func (f *stockFixture) setTenantThreshold(threshold int) {
	f.t.Helper()
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		return repositories.NewTenantSettingsRepo(f.db).Upsert(ctx, &settingsDomain.TenantSettings{
			TenantID: tenant.IDFromContext(ctx), LowStockThreshold: threshold, UpdatedAt: time.Now(),
		})
	}))
}

func (f *stockFixture) list(filter pmsDomain.ProductFilter) (names []string, total int64) {
	f.t.Helper()
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		products, n, err := f.products.List(ctx, filter)
		for _, p := range products {
			names = append(names, p.Name)
		}
		total = n
		return err
	}))
	sort.Strings(names)
	return names, total
}

func TestProductRepo_Integration_StockFilterOutOfStock(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := newStockFixture(t, db, ctx)

	f.product("in-stock", pmsDomain.ProductStatusActive, 50)
	f.product("zero", pmsDomain.ProductStatusActive, 0)
	f.product("no-skus", pmsDomain.ProductStatusActive)
	f.product("negative", pmsDomain.ProductStatusActive, -3)
	// Stock in and then all out again: the ledger sums to 0.
	_, soldOut := f.product("sold-out", pmsDomain.ProductStatusActive, 5)
	f.addStock(soldOut[0], -5)
	// One empty SKU does not make the product out of stock.
	f.product("one-empty-sku", pmsDomain.ProductStatusActive, 0, 4)
	f.product("archived-empty", pmsDomain.ProductStatusArchived, 0)

	names, total := f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelOut})

	require.Equal(t, []string{"negative", "no-skus", "sold-out", "zero"}, names, "archived products are left out")
	require.Equal(t, int64(4), total)
}

func TestProductRepo_Integration_StockFilterLowUsesTenantThreshold(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := newStockFixture(t, db, ctx)

	f.product("at-default", pmsDomain.ProductStatusActive, settingsDomain.DefaultLowStockThreshold)
	f.product("above-default", pmsDomain.ProductStatusActive, settingsDomain.DefaultLowStockThreshold+1)
	f.product("one-left", pmsDomain.ProductStatusActive, 1)
	f.product("empty", pmsDomain.ProductStatusActive, 0)
	// Totals add up across SKUs and warehouses: 6 + 6 = 12, not low at 10.
	f.product("two-skus", pmsDomain.ProductStatusActive, 6, 6)

	names, total := f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow})
	require.Equal(t, []string{"at-default", "one-left"}, names, "never-saved tenant: the default threshold, 0 is out not low")
	require.Equal(t, int64(2), total)

	f.setTenantThreshold(12)
	names, _ = f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow})
	require.Equal(t, []string{"above-default", "at-default", "one-left", "two-skus"}, names)

	f.setTenantThreshold(0)
	names, _ = f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow})
	require.Empty(t, names, "a threshold of 0 never flags low")
}

func TestProductRepo_Integration_StockFilterLowPrefersProductThreshold(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := newStockFixture(t, db, ctx)

	// A weighed product: 800 g left, its own line at 1000 g.
	cheese, _ := f.product("cheese-by-weight", pmsDomain.ProductStatusActive, 800)
	// 5 left but its own line is 2: not low despite the tenant's 10.
	strict, _ := f.product("strict", pmsDomain.ProductStatusActive, 5)
	f.product("tenant-rule", pmsDomain.ProductStatusActive, 5)
	thousand, two := 1000, 2
	f.setOwnThreshold(cheese, &thousand)
	f.setOwnThreshold(strict, &two)

	names, _ := f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow})
	require.Equal(t, []string{"cheese-by-weight", "tenant-rule"}, names)

	f.setOwnThreshold(strict, nil)
	names, _ = f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow})
	require.Equal(t, []string{"cheese-by-weight", "strict", "tenant-rule"}, names, "clearing the override falls back to the tenant's")
}

// The bug this fixes: the inventory screen filtered one page of all
// products in the browser, so page 1 showed only the few low ones that
// happened to be on it. The filter, count and paging are now one query.
func TestProductRepo_Integration_StockFilterPaginatesOverMatchesOnly(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := newStockFixture(t, db, ctx)

	for i := 0; i < 25; i++ {
		f.product("plenty-"+string(rune('a'+i)), pmsDomain.ProductStatusActive, 100)
		f.product("low-"+string(rune('a'+i)), pmsDomain.ProductStatusActive, 3)
	}

	page1, total := f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow, Limit: 20})
	page2, _ := f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow, Limit: 20, Offset: 20})

	require.Equal(t, int64(25), total)
	require.Len(t, page1, 20)
	require.Len(t, page2, 5)
}

func TestProductRepo_Integration_StockFilterCombinesWithSearch(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := newStockFixture(t, db, ctx)

	f.product("Feijão preto", pmsDomain.ProductStatusActive, 2)
	f.product("Feijão carioca", pmsDomain.ProductStatusActive, 200)
	f.product("Farinha", pmsDomain.ProductStatusActive, 2)

	search := "feijao"
	names, total := f.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow, Search: &search})

	require.Equal(t, []string{"Feijão preto"}, names, "same accent-insensitive search as the plain list")
	require.Equal(t, int64(1), total)
}

func TestProductRepo_Integration_StockFilterIsTenantIsolated(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	newStockFixture(t, db, ctxA).product("tenant-a-low", pmsDomain.ProductStatusActive, 2)
	fb := newStockFixture(t, db, ctxB)
	fb.product("tenant-b-low", pmsDomain.ProductStatusActive, 2)
	fb.setTenantThreshold(5)

	names, _ := fb.list(pmsDomain.ProductFilter{StockLevel: pmsDomain.StockLevelLow})

	require.Equal(t, []string{"tenant-b-low"}, names)
}

func TestProductRepo_Integration_LowStockThresholdRoundTripsAndSurvivesUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := newStockFixture(t, db, ctx)
	id, _ := f.product("thresholded", pmsDomain.ProductStatusActive, 1)
	seven := 7
	f.setOwnThreshold(id, &seven)

	var found *pmsDomain.Product
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		p, err := f.products.FindByID(ctx, id)
		if err != nil {
			return err
		}
		// A regular product save (admin form, channel import) must not
		// touch the threshold.
		p.Name = "renamed"
		if err := f.products.Update(ctx, p); err != nil {
			return err
		}
		found, err = f.products.FindByID(ctx, id)
		return err
	}))
	require.NotNil(t, found.LowStockThreshold)
	require.Equal(t, 7, *found.LowStockThreshold)

	f.setOwnThreshold(id, nil)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = f.products.FindByID(ctx, id)
		return err
	}))
	require.Nil(t, found.LowStockThreshold)
}

func TestProductRepo_Integration_SetLowStockThresholdUnknownProduct(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	one := 1
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repositories.NewProductRepo(db).SetLowStockThreshold(ctx, "00000000-0000-4000-8000-000000000000", &one)
	})
	require.ErrorIs(t, err, pmsDomain.ErrProductNotFound)
}
