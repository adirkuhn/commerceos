package pms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type priceListCatalog interface {
	Execute(ctx context.Context, in ListProductsInput) (*ListPOSProductCatalogResult, error)
}

type priceListAvailability interface {
	Execute(ctx context.Context, skuIDs []string) (map[string]invapp.SKUAvailability, error)
}

type priceListSettings interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type priceListCategories interface {
	List(ctx context.Context) ([]pmsDomain.Category, error)
}

type priceListTaxRates interface {
	Rate(ctx context.Context, taxClassID *string, country string) (rateBasisPoints int64, configured bool, err error)
}

// PriceListRow is one sellable SKU with everything needed to price it for
// a discount or a B2B quote. Discounts, profit and totals are computed by
// the caller -- the quote screen edits prices live, so this stays raw data.
type PriceListRow struct {
	ProductID    string
	ProductName  string
	VariantName  string
	SKUID        string
	SKUCode      string
	UnitType     pmsDomain.SKUUnitType
	CurrencyCode string
	// PriceMinorUnits is the regular price; SalePriceMinorUnits only when a
	// sale runs today. Both as stored: tax-inclusive when
	// ListPriceListResult.PricesIncludeTax.
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	// CostPriceMinorUnits is the SKU's average cost, always excluding tax;
	// nil when unknown.
	CostPriceMinorUnits *int64
	AvailableQuantity   int64
	EarliestExpiry      *time.Time
	TaxRateBasisPoints  int64
	// TaxRateConfigured is false when the SKU's VAT class (or, without one,
	// the shop's default class) has no rate for the tax country -- then
	// TaxRateBasisPoints is a 0 nobody chose, worth a warning.
	TaxRateConfigured bool
	// TargetMarkupBasisPoints is the markup (profit / cost) below which a
	// price is too low -- the category override, else the tenant default.
	TargetMarkupBasisPoints int
}

type ListPriceListInput struct {
	Page    int
	PerPage int
	Search  string
	// ProductIDs, when set, restricts the list to these products (a saved
	// quote reloading its lines).
	ProductIDs []string
}

type ListPriceListResult struct {
	Rows []PriceListRow
	// Total counts products, not rows -- pagination is per product.
	Total            int64
	PricesIncludeTax bool
}

// ListPriceListUseCase lists active products' active SKUs with price,
// cost, sellable stock, earliest expiry, VAT rate and target markup.
type ListPriceListUseCase struct {
	db           *gorm.DB
	catalog      priceListCatalog
	availability priceListAvailability
	settings     priceListSettings
	categories   priceListCategories
	taxRates     priceListTaxRates
}

func NewListPriceListUseCase(
	db *gorm.DB,
	catalog priceListCatalog,
	availability priceListAvailability,
	settings priceListSettings,
	categories priceListCategories,
	taxRates priceListTaxRates,
) *ListPriceListUseCase {
	return &ListPriceListUseCase{db: db, catalog: catalog, availability: availability, settings: settings, categories: categories, taxRates: taxRates}
}

func (uc *ListPriceListUseCase) Execute(ctx context.Context, in ListPriceListInput) (*ListPriceListResult, error) {
	active := pmsDomain.ProductStatusActive
	catalogIn := ListProductsInput{Page: in.Page, PerPage: in.PerPage, Search: in.Search, Status: &active}
	if in.ProductIDs != nil {
		ids := in.ProductIDs
		catalogIn.IncludeIDs = &ids
	}
	catalog, err := uc.catalog.Execute(ctx, catalogIn)
	if err != nil {
		return nil, fmt.Errorf("listPriceList: %w", err)
	}

	rows := activeSKURows(catalog.Products, time.Now())
	skuIDs := make([]string, len(rows))
	for i, r := range rows {
		skuIDs[i] = r.row.SKUID
	}
	availability, err := uc.availability.Execute(ctx, skuIDs)
	if err != nil {
		return nil, fmt.Errorf("listPriceList: %w", err)
	}

	result := &ListPriceListResult{Total: catalog.Total, Rows: make([]PriceListRow, len(rows))}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings, err := uc.settings.Get(ctx)
		if err != nil {
			return err
		}
		categories, err := uc.categories.List(ctx)
		if err != nil {
			return err
		}
		categoryByID := make(map[string]*pmsDomain.Category, len(categories))
		for i := range categories {
			categoryByID[categories[i].ID] = &categories[i]
		}
		rates := taxRateCache{resolver: uc.taxRates, country: settings.DefaultTaxCountry, byClass: map[string]resolvedRate{}}

		// A never-saved tenant reads as all zero values, but its default is
		// tax-inclusive prices (the column defaults to true).
		result.PricesIncludeTax = settings.PricesIncludeTax || settings.TenantID == ""
		for i, r := range rows {
			row := r.row
			stock := availability[row.SKUID]
			row.AvailableQuantity, row.EarliestExpiry = stock.Available, stock.EarliestExpiry
			if row.TaxRateBasisPoints, row.TaxRateConfigured, err = rates.get(ctx, r.taxClassID); err != nil {
				return err
			}
			var category *pmsDomain.Category
			if r.categoryID != nil {
				category = categoryByID[*r.categoryID]
			}
			row.TargetMarkupBasisPoints = EffectiveTargetMarkupBasisPoints(category, settings.DefaultTargetMarkupBasisPoints)
			result.Rows[i] = row
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listPriceList: %w", err)
	}
	return result, nil
}

// pendingPriceListRow carries the lookup keys a row still needs resolved.
type pendingPriceListRow struct {
	row        PriceListRow
	taxClassID *string
	categoryID *string
}

func activeSKURows(products []POSProductWithSKUs, today time.Time) []pendingPriceListRow {
	var rows []pendingPriceListRow
	for _, p := range products {
		for _, s := range p.SKUs {
			if s.Status == pmsDomain.SKUStatusInactive {
				continue
			}
			rows = append(rows, pendingPriceListRow{
				row: PriceListRow{
					ProductID: p.ID, ProductName: p.Name, VariantName: s.VariantName,
					SKUID: s.ID, SKUCode: s.SKUCode, UnitType: s.UnitType, CurrencyCode: s.CurrencyCode,
					PriceMinorUnits: s.PriceMinorUnits, SalePriceMinorUnits: s.ActiveSalePrice(today),
					CostPriceMinorUnits: s.CostPriceMinorUnits,
				},
				taxClassID: s.TaxClassID,
				categoryID: p.CategoryID,
			})
		}
	}
	return rows
}

// taxRateCache resolves each tax class once per request -- the resolver
// reloads every class and rate on each call.
type taxRateCache struct {
	resolver priceListTaxRates
	country  string
	byClass  map[string]resolvedRate
}

type resolvedRate struct {
	basisPoints int64
	configured  bool
}

func (c *taxRateCache) get(ctx context.Context, taxClassID *string) (int64, bool, error) {
	key := ""
	if taxClassID != nil {
		key = *taxClassID
	}
	if r, ok := c.byClass[key]; ok {
		return r.basisPoints, r.configured, nil
	}
	bp, configured, err := c.resolver.Rate(ctx, taxClassID, c.country)
	if err != nil {
		return 0, false, err
	}
	c.byClass[key] = resolvedRate{bp, configured}
	return bp, configured, nil
}
