package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type PreviewProductMergeInput struct {
	SourceProductID string
	TargetProductID string
	// Pairs is the user's SKU matching so far; empty asks for a suggestion.
	Pairs []pmsDomain.SKUMergePair
}

type MergePreviewSKU struct {
	ID                  string
	SKUCode             string
	Barcode             string
	OnHand              int64
	CostPriceMinorUnits *int64
}

type MergePreviewProduct struct {
	ID   string
	Name string
	SKUs []MergePreviewSKU
}

// MergePreviewTarget is an existing SKU after the merge.
type MergePreviewTarget struct {
	SKUID       string
	OnHandAfter int64
	CostBefore  *int64
	CostAfter   *int64
}

type ProductMergePreview struct {
	Source MergePreviewProduct
	Target MergePreviewProduct
	// Pairs is the given mapping, or a suggestion: the 1:1 match when both
	// products have one SKU, else the SKUs whose barcode matches exactly.
	Pairs []pmsDomain.SKUMergePair
	// Targets is filled once Pairs matches every duplicate SKU.
	Targets []MergePreviewTarget
}

// PreviewProductMergeUseCase shows what MergeProductsUseCase would do,
// without changing anything: both products' SKUs with stock and cost, a
// suggested SKU matching, and each existing SKU's stock and cost after.
type PreviewProductMergeUseCase struct {
	db       *gorm.DB
	products mergeProductStore
	variants mergeVariantLister
	skus     mergeSKULister
	onHand   mergeOnHand
}

func NewPreviewProductMergeUseCase(db *gorm.DB, products mergeProductStore, variants mergeVariantLister, skus mergeSKULister, onHand mergeOnHand) *PreviewProductMergeUseCase {
	return &PreviewProductMergeUseCase{db: db, products: products, variants: variants, skus: skus, onHand: onHand}
}

func (uc *PreviewProductMergeUseCase) Execute(ctx context.Context, in PreviewProductMergeInput) (*ProductMergePreview, error) {
	if in.SourceProductID == in.TargetProductID {
		return nil, fmt.Errorf("previewProductMerge: %w", pmsDomain.ErrMergeIntoSelf)
	}
	var preview ProductMergePreview
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		bySKUID := make(map[string]pmsDomain.SKU)
		onHand := make(map[string]int64)
		load := func(productID string) (MergePreviewProduct, []pmsDomain.SKU, error) {
			product, err := uc.products.FindByID(ctx, productID)
			if err != nil {
				return MergePreviewProduct{}, nil, err
			}
			skus, err := listProductSKUs(ctx, uc.variants, uc.skus, productID)
			if err != nil {
				return MergePreviewProduct{}, nil, err
			}
			out := MergePreviewProduct{ID: product.ID, Name: product.Name, SKUs: make([]MergePreviewSKU, len(skus))}
			for i, s := range skus {
				qty, err := sumOnHand(ctx, uc.onHand, s.ID)
				if err != nil {
					return MergePreviewProduct{}, nil, err
				}
				bySKUID[s.ID], onHand[s.ID] = s, qty
				out.SKUs[i] = MergePreviewSKU{ID: s.ID, SKUCode: s.SKUCode, Barcode: s.Barcode, OnHand: qty, CostPriceMinorUnits: s.CostPriceMinorUnits}
			}
			return out, skus, nil
		}
		var sourceSKUs, targetSKUs []pmsDomain.SKU
		var err error
		if preview.Source, sourceSKUs, err = load(in.SourceProductID); err != nil {
			return err
		}
		if preview.Target, targetSKUs, err = load(in.TargetProductID); err != nil {
			return err
		}

		preview.Pairs = in.Pairs
		if len(preview.Pairs) == 0 {
			preview.Pairs = suggestMergePairs(sourceSKUs, targetSKUs)
		}
		merge, err := pmsDomain.NewProductMerge(in.SourceProductID, in.TargetProductID, sourceSKUs, targetSKUs, preview.Pairs)
		if err != nil {
			if len(in.Pairs) > 0 {
				return err // the user's own mapping is wrong: say so
			}
			return nil // a partial suggestion: no projection yet
		}
		index := make(map[string]int)
		for _, step := range foldMergePairs(merge.Pairs, bySKUID, onHand) {
			target := MergePreviewTarget{SKUID: step.pair.TargetSKUID, OnHandAfter: step.qtyAfter, CostAfter: step.costAfter}
			if i, seen := index[target.SKUID]; seen {
				target.CostBefore = preview.Targets[i].CostBefore
				preview.Targets[i] = target
				continue
			}
			target.CostBefore = step.costBefore
			index[target.SKUID] = len(preview.Targets)
			preview.Targets = append(preview.Targets, target)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("previewProductMerge: %w", err)
	}
	return &preview, nil
}

func suggestMergePairs(sourceSKUs, targetSKUs []pmsDomain.SKU) []pmsDomain.SKUMergePair {
	if pairs, ok := pmsDomain.DefaultSKUMergePairs(sourceSKUs, targetSKUs); ok {
		return pairs
	}
	byBarcode := make(map[string]string)
	for _, t := range targetSKUs {
		if t.Barcode != "" {
			byBarcode[t.Barcode] = t.ID
		}
	}
	var pairs []pmsDomain.SKUMergePair
	for _, s := range sourceSKUs {
		if id, ok := byBarcode[s.Barcode]; ok && s.Barcode != "" {
			pairs = append(pairs, pmsDomain.SKUMergePair{SourceSKUID: s.ID, TargetSKUID: id})
		}
	}
	return pairs
}
