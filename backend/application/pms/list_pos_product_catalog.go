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

type mediaByProductIDsRepo interface {
	ListByProductIDs(ctx context.Context, productIDs []string) ([]pmsDomain.Media, error)
}

type barcodeAliasesBySKUIDsRepo interface {
	ListBySKUIDs(ctx context.Context, skuIDs []string) ([]pmsDomain.BarcodeAlias, error)
}

type scaleCodeAliasesBySKUIDsRepo interface {
	ListBySKUIDs(ctx context.Context, skuIDs []string) ([]pmsDomain.ScaleCodeAlias, error)
}

// POSBarcodeAlias is one of a SKU's alternate barcodes -- see
// pmsDomain.BarcodeAlias.
type POSBarcodeAlias struct {
	Code         string
	UnitsPerScan int64
}

// POSProductSKU is a SKU as seen by an external POS integration -- the SKU
// itself plus its parent variant's name, since a POS has no separate
// variant concept to display it under. SaleEndDate comes along for free
// (embedded in pmsDomain.SKU) -- a POS needs it to know when a discounted
// SalePriceMinorUnits stops applying, same as the admin catalog UI.
type POSProductSKU struct {
	pmsDomain.SKU
	VariantName string
	// BarcodeAliases are this SKU's alternate barcodes -- either a
	// case/box multiplier or a plain alternate unit barcode
	// (UnitsPerScan 1) -- e.g. the same product repackaged by two
	// suppliers, each with its own EAN. Empty for a SKU with none.
	BarcodeAliases []POSBarcodeAlias
	// ScaleCodeAliases are this SKU's additional scale codes, alongside
	// its own primary SKU.ScaleCode -- for a tenant with more than one
	// physical scale. Empty for a SKU with none.
	ScaleCodeAliases []string
}

// POSProductWithSKUs is a catalog product plus every SKU across all of its
// variants, flattened -- the shape an external POS needs to match/import
// each purchasable item (see ListPOSProductCatalogUseCase). ImageURL is nil
// when the product has no media -- otherwise the lowest-Position (see
// Media's doc comment) image, same "first = featured" convention the admin
// frontend already uses (ProductCandidateRow.tsx's media[0]).
type POSProductWithSKUs struct {
	pmsDomain.Product
	SKUs     []POSProductSKU
	ImageURL *string
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
	db                 *gorm.DB
	listProducts       *ListProductsUseCase
	variantRepo        variantsByProductIDsRepo
	skuRepo            skusByVariantIDsRepo
	mediaRepo          mediaByProductIDsRepo
	barcodeAliasRepo   barcodeAliasesBySKUIDsRepo
	scaleCodeAliasRepo scaleCodeAliasesBySKUIDsRepo
}

func NewListPOSProductCatalogUseCase(
	db *gorm.DB,
	listProducts *ListProductsUseCase,
	variantRepo variantsByProductIDsRepo,
	skuRepo skusByVariantIDsRepo,
	mediaRepo mediaByProductIDsRepo,
	barcodeAliasRepo barcodeAliasesBySKUIDsRepo,
	scaleCodeAliasRepo scaleCodeAliasesBySKUIDsRepo,
) *ListPOSProductCatalogUseCase {
	return &ListPOSProductCatalogUseCase{
		db: db, listProducts: listProducts, variantRepo: variantRepo, skuRepo: skuRepo, mediaRepo: mediaRepo,
		barcodeAliasRepo: barcodeAliasRepo, scaleCodeAliasRepo: scaleCodeAliasRepo,
	}
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
		skuIDs := make([]string, len(skus))
		for i, s := range skus {
			skusByVariant[s.VariantID] = append(skusByVariant[s.VariantID], s)
			skuIDs[i] = s.ID
		}

		barcodeAliases, err := uc.barcodeAliasRepo.ListBySKUIDs(ctx, skuIDs)
		if err != nil {
			return err
		}
		barcodeAliasesBySKU := make(map[string][]POSBarcodeAlias, len(skuIDs))
		for _, a := range barcodeAliases {
			barcodeAliasesBySKU[a.SKUID] = append(barcodeAliasesBySKU[a.SKUID], POSBarcodeAlias{Code: a.Code, UnitsPerScan: a.UnitsPerScan})
		}

		scaleCodeAliases, err := uc.scaleCodeAliasRepo.ListBySKUIDs(ctx, skuIDs)
		if err != nil {
			return err
		}
		scaleCodeAliasesBySKU := make(map[string][]string, len(skuIDs))
		for _, a := range scaleCodeAliases {
			scaleCodeAliasesBySKU[a.SKUID] = append(scaleCodeAliasesBySKU[a.SKUID], a.Code)
		}

		media, err := uc.mediaRepo.ListByProductIDs(ctx, productIDs)
		if err != nil {
			return err
		}
		// First entry per product wins -- ListByProductIDs orders by
		// Position asc, so this is "the featured image" (see
		// POSProductWithSKUs's doc comment).
		featuredImageByProduct := make(map[string]string, len(productIDs))
		for _, m := range media {
			if m.ProductID == nil {
				continue
			}
			if _, seen := featuredImageByProduct[*m.ProductID]; seen {
				continue
			}
			featuredImageByProduct[*m.ProductID] = m.URL
		}

		result.Products = make([]POSProductWithSKUs, len(productsResult.Products))
		for i, p := range productsResult.Products {
			entry := POSProductWithSKUs{Product: p}
			if url, ok := featuredImageByProduct[p.ID]; ok {
				entry.ImageURL = &url
			}
			for _, v := range variantsByProduct[p.ID] {
				for _, s := range skusByVariant[v.ID] {
					entry.SKUs = append(entry.SKUs, POSProductSKU{
						SKU: s, VariantName: v.Name,
						BarcodeAliases: barcodeAliasesBySKU[s.ID], ScaleCodeAliases: scaleCodeAliasesBySKU[s.ID],
					})
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
