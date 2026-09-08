package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
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
	db           *gorm.DB
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	listCatalog  listPOSProductCatalogUC
	productRepo  pmsDomain.ProductRepository
	categoryRepo pmsDomain.CategoryRepository
}

func NewListProductsUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	listCatalog listPOSProductCatalogUC,
	productRepo pmsDomain.ProductRepository,
	categoryRepo pmsDomain.CategoryRepository,
) *ListProductsUseCase {
	return &ListProductsUseCase{
		db: db, channelRepo: channelRepo, overrideRepo: overrideRepo, listCatalog: listCatalog,
		productRepo: productRepo, categoryRepo: categoryRepo,
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
		var includeIDs, excludeIDs []string
		for _, o := range overrides {
			if o.Included {
				includeIDs = append(includeIDs, o.ProductID)
			} else {
				excludeIDs = append(excludeIDs, o.ProductID)
			}
		}

		active := pmsDomain.ProductStatusActive
		pmsInput = apppms.ListProductsInput{
			Page: in.Page, PerPage: in.PerPage, Search: in.Search,
			AndCategoryID: in.CategoryID, ExcludeIDs: excludeIDs, Status: &active,
		}
		switch channel.EnrollmentMode {
		case channelsDomain.EnrollmentModeAll:
			// No OR-group constraint needed -- everything active is
			// enrolled by default, ExcludeIDs above still applies.
		case channelsDomain.EnrollmentModeCategory:
			pmsInput.CategoryID = channel.EnrollmentCategoryID
			pmsInput.IncludeIDs = &includeIDs
		case channelsDomain.EnrollmentModeTag:
			pmsInput.Tag = &channel.EnrollmentTag
			pmsInput.IncludeIDs = &includeIDs
		case channelsDomain.EnrollmentModeManual:
			pmsInput.IncludeIDs = &includeIDs
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
	for i, p := range catalog.Products {
		summary := ProductSummary{Product: p.Product, ImageURL: p.ImageURL}
		if len(p.SKUs) > 0 {
			sku := p.SKUs[0]
			price, currency := channel.EffectivePriceMinorUnits(sku.PriceMinorUnits), sku.CurrencyCode
			summary.PriceMinorUnits, summary.CurrencyCode = &price, &currency
			if sku.SalePriceMinorUnits != nil {
				salePrice := channel.EffectivePriceMinorUnits(*sku.SalePriceMinorUnits)
				summary.SalePriceMinorUnits = &salePrice
			}
		}
		result.Products[i] = summary
	}
	return result, nil
}
