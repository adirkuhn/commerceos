package pms

import (
	"context"
	"errors"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const maxDuplicateNameAttempts = 50

// DuplicateProductUseCase clones a product's catalog data -- name, category,
// tags, variants, and SKUs (pricing/physical attributes) -- into a
// brand-new product. Two deliberate departures from a plain copy:
//
//   - Status is always forced to draft, and this use case never raises
//     pms.product.changed (unlike CreateProductUseCase). Both matter for the
//     same reason: EnrollProductInChannelsUseCase reacts to that event with
//     no status filter at all, so a duplicate that fired the normal event
//     could auto-enroll into a matching SalesChannel and push to Woo before
//     anyone has looked at it. Skipping the event keeps a duplicate
//     completely inert until the user explicitly saves it (which raises the
//     event normally, through UpdateProductUseCase).
//   - Barcode and cost-price fields are dropped rather than copied: a
//     duplicated barcode would collide with a real, scannable physical
//     item, and cost price has no basis until this duplicate has its own
//     purchase/batch history.
//
// Media (images) is not duplicated -- the clone starts with none.
// CaseOfSKUID/PortionedFromSKUID are copied through as-is (pointing at
// whatever SKU the source pointed at, not a same-duplicate sibling) -- the
// common case is that relationship pointing at an unrelated product's bulk
// SKU, which remains correct; a source product that references its own
// sibling SKU this way is not remapped to the new duplicate's SKUs.
type DuplicateProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
}

func NewDuplicateProductUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
) *DuplicateProductUseCase {
	return &DuplicateProductUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo}
}

func (uc *DuplicateProductUseCase) Execute(ctx context.Context, sourceProductID string) (*ProductDetail, error) {
	var result ProductDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		source, err := uc.productRepo.FindByID(ctx, sourceProductID)
		if err != nil {
			return err
		}
		variants, err := uc.variantRepo.ListByProduct(ctx, sourceProductID)
		if err != nil {
			return err
		}

		name, slug, err := uc.uniqueCopyName(ctx, source.Name)
		if err != nil {
			return err
		}

		now := time.Now()
		newProduct := pmsDomain.Product{
			TenantID:                    tenant.IDFromContext(ctx),
			CategoryID:                  source.CategoryID,
			Name:                        name,
			Slug:                        slug,
			Description:                 source.Description,
			Status:                      pmsDomain.ProductStatusDraft,
			HasVariants:                 source.HasVariants,
			Tags:                        append([]string(nil), source.Tags...),
			SyncVariantsAsWooVariations: source.SyncVariantsAsWooVariations,
			CreatedAt:                   now,
			UpdatedAt:                   now,
		}
		if err := uc.productRepo.Create(ctx, &newProduct); err != nil {
			return err
		}
		result.Product = newProduct

		result.Variants = make([]VariantDetail, len(variants))
		for i, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}

			newVariant := pmsDomain.Variant{
				TenantID:   newProduct.TenantID,
				ProductID:  newProduct.ID,
				Name:       v.Name,
				Position:   v.Position,
				Attributes: append([]pmsDomain.VariantAttribute(nil), v.Attributes...),
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if err := uc.variantRepo.Create(ctx, &newVariant); err != nil {
				return err
			}

			newSKUs := make([]pmsDomain.SKU, len(skus))
			for j, s := range skus {
				skuCode, err := uc.uniqueSKUCode(ctx, s.SKUCode)
				if err != nil {
					return err
				}
				newSKUs[j] = pmsDomain.SKU{
					TenantID:  newProduct.TenantID,
					VariantID: newVariant.ID,
					SKUCode:   skuCode,
					// Barcode deliberately left blank -- see doc comment.
					PriceMinorUnits:     s.PriceMinorUnits,
					SalePriceMinorUnits: s.SalePriceMinorUnits,
					SaleEndDate:         s.SaleEndDate,
					CurrencyCode:        s.CurrencyCode,
					// CostPriceMinorUnits/CostPriceSource deliberately left
					// zero-value -- no cost basis for this duplicate yet.
					WeightGrams:        s.WeightGrams,
					LengthMM:           s.LengthMM,
					WidthMM:            s.WidthMM,
					HeightMM:           s.HeightMM,
					IsVirtual:          s.IsVirtual,
					TracksExpiry:       s.TracksExpiry,
					CaseOfSKUID:        s.CaseOfSKUID,
					UnitsPerCase:       s.UnitsPerCase,
					UnitType:           s.UnitType,
					PortionedFromSKUID: s.PortionedFromSKUID,
					TaxClassID:         s.TaxClassID,
					// WooTaxClassSlug/WooTaxStatus deliberately left nil --
					// provenance of a Woo import, not this fresh duplicate.
					Status:    s.Status,
					CreatedAt: now,
					UpdatedAt: now,
				}
				if err := uc.skuRepo.Create(ctx, &newSKUs[j]); err != nil {
					return err
				}
			}
			result.Variants[i] = VariantDetail{Variant: newVariant, SKUs: newSKUs}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("duplicateProduct: %w", err)
	}
	return &result, nil
}

// uniqueCopyName tries "{name} (Copy)", "{name} (Copy 2)", ... until it
// finds a slug with no existing collision.
func (uc *DuplicateProductUseCase) uniqueCopyName(ctx context.Context, sourceName string) (name, slug string, err error) {
	for i := 1; i <= maxDuplicateNameAttempts; i++ {
		candidate := sourceName + " (Copy)"
		if i > 1 {
			candidate = fmt.Sprintf("%s (Copy %d)", sourceName, i)
		}
		candidateSlug := Slugify(candidate)
		_, err := uc.productRepo.FindBySlug(ctx, candidateSlug)
		if errors.Is(err, pmsDomain.ErrProductNotFound) {
			return candidate, candidateSlug, nil
		}
		if err != nil {
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("could not find a unique product name after %d attempts", maxDuplicateNameAttempts)
}

// uniqueSKUCode mirrors uniqueCopyName for SKU codes.
func (uc *DuplicateProductUseCase) uniqueSKUCode(ctx context.Context, sourceCode string) (string, error) {
	for i := 1; i <= maxDuplicateNameAttempts; i++ {
		candidate := sourceCode + "-COPY"
		if i > 1 {
			candidate = fmt.Sprintf("%s-COPY-%d", sourceCode, i)
		}
		_, err := uc.skuRepo.FindByCode(ctx, candidate)
		if errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("could not find a unique SKU code after %d attempts", maxDuplicateNameAttempts)
}
