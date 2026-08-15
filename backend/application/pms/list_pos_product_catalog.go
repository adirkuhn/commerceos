package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// variantsByProductIDsRepo and skusByVariantIDsRepo are narrow, point-of-use
// interfaces (ISP) -- satisfied by *repositories.VariantRepo/*SKURepo without
// widening the shared pmsDomain.VariantRepository/SKURepository interfaces,
// which would otherwise force every existing fake across the codebase to
// grow these two methods for no benefit to them.
type variantsByProductIDsRepo interface {
	ListByProductIDs(ctx context.Context, productIDs []string) ([]pmsDomain.Variant, error)
}

type skusByVariantIDsRepo interface {
	ListByVariantIDs(ctx context.Context, variantIDs []string) ([]pmsDomain.SKU, error)
}

// POSProductSKU is a SKU as seen by an external POS integration -- the SKU
// itself plus its parent variant's name, since a POS has no separate
// variant concept to display it under.
type POSProductSKU struct {
	pmsDomain.SKU
	VariantName string
}

// POSProductWithSKUs is a catalog product plus every SKU across all of its
// variants, flattened -- the shape an external POS needs to match/import
// each purchasable item (see ListPOSProductCatalogUseCase).
type POSProductWithSKUs struct {
	pmsDomain.Product
	SKUs []POSProductSKU
}

type ListPOSProductCatalogResult struct {
	Products []POSProductWithSKUs
	Total    int64
}

// ListPOSProductCatalogUseCase wraps ListProductsUseCase (unchanged, still
// used as-is by the admin /pms/products endpoint) and bulk-loads each
// returned product's SKUs -- two extra queries per page rather than one per
// product, regardless of page size.
type ListPOSProductCatalogUseCase struct {
	db           *gorm.DB
	listProducts *ListProductsUseCase
	variantRepo  variantsByProductIDsRepo
	skuRepo      skusByVariantIDsRepo
}

func NewListPOSProductCatalogUseCase(
	db *gorm.DB,
	listProducts *ListProductsUseCase,
	variantRepo variantsByProductIDsRepo,
	skuRepo skusByVariantIDsRepo,
) *ListPOSProductCatalogUseCase {
	return &ListPOSProductCatalogUseCase{db: db, listProducts: listProducts, variantRepo: variantRepo, skuRepo: skuRepo}
}

func (uc *ListPOSProductCatalogUseCase) Execute(ctx context.Context, in ListProductsInput) (*ListPOSProductCatalogResult, error) {
	productsResult, err := uc.listProducts.Execute(ctx, in)
	if err != nil {
		return nil, err
	}

	result := &ListPOSProductCatalogResult{Total: productsResult.Total}
	if len(productsResult.Products) == 0 {
		return result, nil
	}

	productIDs := make([]string, len(productsResult.Products))
	for i, p := range productsResult.Products {
		productIDs[i] = p.ID
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variants, err := uc.variantRepo.ListByProductIDs(ctx, productIDs)
		if err != nil {
			return err
		}

		variantIDs := make([]string, len(variants))
		variantsByProduct := make(map[string][]pmsDomain.Variant, len(productIDs))
		for i, v := range variants {
			variantIDs[i] = v.ID
			variantsByProduct[v.ProductID] = append(variantsByProduct[v.ProductID], v)
		}

		skus, err := uc.skuRepo.ListByVariantIDs(ctx, variantIDs)
		if err != nil {
			return err
		}

		skusByVariant := make(map[string][]pmsDomain.SKU, len(variantIDs))
		for _, s := range skus {
			skusByVariant[s.VariantID] = append(skusByVariant[s.VariantID], s)
		}

		result.Products = make([]POSProductWithSKUs, len(productsResult.Products))
		for i, p := range productsResult.Products {
			entry := POSProductWithSKUs{Product: p}
			for _, v := range variantsByProduct[p.ID] {
				for _, s := range skusByVariant[v.ID] {
					entry.SKUs = append(entry.SKUs, POSProductSKU{SKU: s, VariantName: v.Name})
				}
			}
			result.Products[i] = entry
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listPOSProductCatalog: %w", err)
	}
	return result, nil
}
