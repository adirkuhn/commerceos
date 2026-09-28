package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ProductSummary is one product as shown on a storefront listing page --
// price/currency are the first SKU's (SKUs[0]), same "first variant is the
// representative one for a list thumbnail/price" convention
// infrastructure/handlers/pms's toGenProductListItem already uses, not a
// computed minimum across variants -- nil when the product has no SKU yet.
type ProductSummary struct {
	pmsDomain.Product
	ImageURL            *string
	PriceMinorUnits     *int64
	SalePriceMinorUnits *int64
	CurrencyCode        *string
}

type ListProductsInput struct {
	CategoryID *string
	// ProductIDs narrows to exactly this list of product ids, on top of
	// "enrolled on this channel" -- a merchant's manually-curated
	// hand-picked product list (e.g. a storefront Product Grid block)
	// uses this instead of CategoryID. nil means unconstrained; a non-nil
	// empty slice means "match nothing" (see pms.ProductFilter.AndIncludeIDs'
	// own doc comment) -- callers that never set it (every other listing
	// use, e.g. the /products page) leave it nil.
	ProductIDs []string
	// ExcludeProductIDs removes these from the result regardless of every
	// other filter -- combines with the channel's own manual-override
	// excludes (below), same as pms.ProductFilter.ExcludeIDs' own doc
	// comment. A merchant's Featured Products block uses this for "this
	// category, minus these specific ones."
	ExcludeProductIDs []string
	// OrderByIDs keeps results in this order across pages (a collection's
	// own sort); nil keeps the default order.
	OrderByIDs []string
	Search     string
	Page       int
	PerPage    int
}

// CategoryFacet is one category a shopper could switch to from the
// current listing, and how many enrolled products they'd see there --
// counted against the same enrollment scoping and Search text as the
// listing itself, but ignoring any currently-selected browse category
// (see pms.ProductRepository.CountByCategory's doc comment), so every
// facet option's count stays meaningful regardless of which one is active.
type CategoryFacet struct {
	CategoryID   string
	CategoryName string
	Count        int64
}

type ListProductsResult struct {
	Products       []ProductSummary
	Total          int64
	CategoryFacets []CategoryFacet
}

// listPOSProductCatalogUC is *application/pms.ListPOSProductCatalogUseCase
// narrowed to point of use (ISP) -- reused as-is rather than reimplementing
// its bulk variant/SKU/featured-image loading (see that type's own doc
// comment for why it's already N+1-safe).
type listPOSProductCatalogUC interface {
	Execute(ctx context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error)
}

// ListProductsUseCase lists one internal sales channel's enrolled catalog,
// exactly paginated -- it translates the channel's own EnrollmentMode into
// pms.ProductFilter's generic OR-group/AndCategoryID primitives (see that
// type's doc comment), rather than fetching a page and dropping
// unenrolled rows in Go, which would silently under-fill pages. Reuses
// application/pms's existing product+SKU+image bulk-loader
// (ListPOSProductCatalogUseCase) for the actual row shape -- pms stays the
// single source of truth for "what a product+its SKUs look like",
// application/storefront only ever adds channel-enrollment scoping on top.
type ListProductsUseCase struct {
	db                   *gorm.DB
	channelRepo          channelsDomain.SalesChannelRepository
	overrideRepo         channelsDomain.SalesChannelProductOverrideRepository
	listCatalog          listPOSProductCatalogUC
	productRepo          pmsDomain.ProductRepository
	categoryRepo         pmsDomain.CategoryRepository
	skuCurrencyPriceRepo pmsDomain.SKUCurrencyPriceRepository
	fxRepo               fxDomain.ExchangeRateRepository
}

func NewListProductsUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	listCatalog listPOSProductCatalogUC,
	productRepo pmsDomain.ProductRepository,
	categoryRepo pmsDomain.CategoryRepository,
	skuCurrencyPriceRepo pmsDomain.SKUCurrencyPriceRepository,
	fxRepo fxDomain.ExchangeRateRepository,
) *ListProductsUseCase {
	return &ListProductsUseCase{
		db: db, channelRepo: channelRepo, overrideRepo: overrideRepo, listCatalog: listCatalog,
		productRepo: productRepo, categoryRepo: categoryRepo,
		skuCurrencyPriceRepo: skuCurrencyPriceRepo, fxRepo: fxRepo,
	}
}

func (uc *ListProductsUseCase) Execute(ctx context.Context, salesChannelID string, in ListProductsInput) (*ListProductsResult, error) {
	var pmsInput apppms.ListProductsInput
	var channel *channelsDomain.SalesChannel
	var categoryFacets []CategoryFacet
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		channel, err = findInternalChannel(ctx, uc.channelRepo, salesChannelID)
		if err != nil {
			return err
		}
		overrides, err := uc.overrideRepo.ListByChannel(ctx, salesChannelID)
		if err != nil {
			return err
		}
		scope := channelEnrollment(channel, overrides)
		excludeIDs := append(scope.excludeIDs, in.ExcludeProductIDs...)

		active := pmsDomain.ProductStatusActive
		pmsInput = apppms.ListProductsInput{
			Page: in.Page, PerPage: in.PerPage, Search: in.Search,
			AndCategoryID: in.CategoryID, ExcludeIDs: excludeIDs, Status: &active, OrderByIDs: in.OrderByIDs,
			CategoryID: scope.categoryID, Tag: scope.tag, IncludeIDs: scope.includeIDs,
		}
		if in.ProductIDs != nil {
			pmsInput.AndIncludeIDs = &in.ProductIDs
		}

		var searchPtr *string
		if in.Search != "" {
			searchPtr = &in.Search
		}
		counts, err := uc.productRepo.CountByCategory(ctx, pmsDomain.ProductFilter{
			CategoryID: pmsInput.CategoryID, Tag: pmsInput.Tag, IncludeIDs: pmsInput.IncludeIDs,
			ExcludeIDs: pmsInput.ExcludeIDs, Status: pmsInput.Status, Search: searchPtr,
		})
		if err != nil {
			return err
		}
		categories, err := uc.categoryRepo.List(ctx)
		if err != nil {
			return err
		}
		for _, cat := range categories {
			if count, ok := counts[cat.ID]; ok && count > 0 {
				categoryFacets = append(categoryFacets, CategoryFacet{CategoryID: cat.ID, CategoryName: cat.Name, Count: count})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listProducts: %w", err)
	}

	catalog, err := uc.listCatalog.Execute(ctx, pmsInput)
	if err != nil {
		return nil, fmt.Errorf("listProducts: %w", err)
	}

	result := &ListProductsResult{Total: catalog.Total, Products: make([]ProductSummary, len(catalog.Products)), CategoryFacets: categoryFacets}
	// A separate transaction from the one above -- listCatalog.Execute (a
	// full use case) already opened and closed its own in between, and this
	// codebase never nests tenant.WithTenant calls.
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for i, p := range catalog.Products {
			summary := ProductSummary{Product: p.Product, ImageURL: p.ImageURL}
			if len(p.SKUs) > 0 {
				sku := p.SKUs[0]
				resolved, err := apppms.ResolveSKUPrice(ctx, sku.SKU, channel.CurrencyCode, uc.skuCurrencyPriceRepo, uc.fxRepo)
				if err != nil {
					return err
				}
				price, currency := channel.EffectivePriceMinorUnits(resolved.PriceMinorUnits), resolved.CurrencyCode
				summary.PriceMinorUnits, summary.CurrencyCode = &price, &currency
				if resolved.SalePriceMinorUnits != nil {
					salePrice := channel.EffectivePriceMinorUnits(*resolved.SalePriceMinorUnits)
					summary.SalePriceMinorUnits = &salePrice
				}
			}
			result.Products[i] = summary
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listProducts: %w", err)
	}
	return result, nil
}
