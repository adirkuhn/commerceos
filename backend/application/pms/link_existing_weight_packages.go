package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// LinkSkipReason says why a variant's SKU was not linked as a package.
type LinkSkipReason string

const (
	// LinkSkipNotAScaleLabel: neither barcode nor SKU code is a scale label.
	LinkSkipNotAScaleLabel LinkSkipReason = "not_a_scale_label"
	// LinkSkipOtherScaleCode: a scale label, but for another scale code.
	LinkSkipOtherScaleCode LinkSkipReason = "other_scale_code"
	// LinkSkipLinkedToAnotherPool: already a package of a different pool.
	LinkSkipLinkedToAnotherPool LinkSkipReason = "linked_to_another_pool"
	// LinkSkipSeveralSKUs: its variant holds more than one SKU -- a
	// package must be its variant's only SKU to be listed on a channel.
	LinkSkipSeveralSKUs LinkSkipReason = "several_skus"
)

type LinkExistingWeightPackagesInput struct {
	BulkSKUID string
	// DryRun previews what would be linked without changing anything.
	DryRun bool
}

type LinkedWeightPackage struct {
	SKU         pmsDomain.SKU
	VariantName string
	WeightGrams int64
	// PriceMinorUnits is the package's price after linking -- always the
	// pool's rule price (pmsDomain.PackagePriceMinorUnits); PriceChanged
	// says it differs from what the package had.
	PriceMinorUnits int64
	PriceChanged    bool
}

type SkippedWeightPackage struct {
	SKU         pmsDomain.SKU
	VariantName string
	Reason      LinkSkipReason
}

type LinkExistingWeightPackagesResult struct {
	Linked  []LinkedWeightPackage
	Skipped []SkippedWeightPackage
	// Errors names each package whose update failed -- one failure never
	// stops the rest.
	Errors []string
}

type skuUpdaterForLink interface {
	Execute(ctx context.Context, in UpdateSKUInput) (*pmsDomain.SKU, error)
}

// LinkExistingWeightPackagesUseCase adopts packages that were made by hand
// before the weight pool existed -- variants of the pool's product whose
// single SKU carries the pool's scale label (e.g. "~ 380g", barcode
// 2994721003809) -- as the pool's packages: PortionedFromSKUID set, weight
// read from the label, priced by the pool's rule. From then on "Split into packages"
// restocks them (see PackWeightPortionsUseCase) instead of creating
// duplicates, and their existing channel listings and barcodes stay.
type LinkExistingWeightPackagesUseCase struct {
	db        *gorm.DB
	skus      skuReaderForPack
	variants  variantReaderForPack
	updateSKU skuUpdaterForLink
}

func NewLinkExistingWeightPackagesUseCase(db *gorm.DB, skus skuReaderForPack, variants variantReaderForPack, updateSKU skuUpdaterForLink) *LinkExistingWeightPackagesUseCase {
	return &LinkExistingWeightPackagesUseCase{db: db, skus: skus, variants: variants, updateSKU: updateSKU}
}

func (uc *LinkExistingWeightPackagesUseCase) Execute(ctx context.Context, in LinkExistingWeightPackagesInput) (*LinkExistingWeightPackagesResult, error) {
	var bulk pmsDomain.SKU
	result := &LinkExistingWeightPackagesResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skus.FindByID(ctx, in.BulkSKUID)
		if err != nil {
			return err
		}
		if !isScaleSoldWeightPool(*sku) {
			return pmsDomain.ErrBulkSKUNotWeightPool
		}
		bulk = *sku
		poolVariant, err := uc.variants.FindByID(ctx, bulk.VariantID)
		if err != nil {
			return err
		}
		variants, err := uc.variants.ListByProduct(ctx, poolVariant.ProductID)
		if err != nil {
			return err
		}
		for _, v := range variants {
			if v.ID == bulk.VariantID {
				continue
			}
			skus, err := uc.skus.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			classifyPackageCandidates(bulk, v.Name, skus, result)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("linkExistingWeightPackages: %w", err)
	}
	if in.DryRun {
		return result, nil
	}

	linked := result.Linked[:0]
	for _, p := range result.Linked {
		if _, err := uc.updateSKU.Execute(ctx, packageUpdate(bulk, p)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", p.SKU.SKUCode, err))
			continue
		}
		linked = append(linked, p)
	}
	result.Linked = linked
	return result, nil
}

// classifyPackageCandidates sorts one variant's SKUs into result.Linked or
// result.Skipped; packages already linked to this pool are left out.
func classifyPackageCandidates(bulk pmsDomain.SKU, variantName string, skus []pmsDomain.SKU, result *LinkExistingWeightPackagesResult) {
	skip := func(s pmsDomain.SKU, r LinkSkipReason) {
		result.Skipped = append(result.Skipped, SkippedWeightPackage{SKU: s, VariantName: variantName, Reason: r})
	}
	for _, s := range skus {
		if s.PortionedFromSKUID != nil {
			if *s.PortionedFromSKUID != bulk.ID {
				skip(s, LinkSkipLinkedToAnotherPool)
			}
			continue
		}
		if len(skus) > 1 {
			skip(s, LinkSkipSeveralSKUs)
			continue
		}
		weight, reason := scaleLabelWeight(bulk, s)
		if reason != "" {
			skip(s, reason)
			continue
		}
		price := pmsDomain.PackagePriceMinorUnits(*bulk.PricePerKiloMinorUnits, weight)
		changed := price != s.PriceMinorUnits || s.PricingMode != pmsDomain.SKUPricingModeFixed
		result.Linked = append(result.Linked, LinkedWeightPackage{
			SKU: s, VariantName: variantName, WeightGrams: weight, PriceMinorUnits: price, PriceChanged: changed,
		})
	}
}

// scaleLabelWeight reads the package weight from its barcode, else its SKU
// code (hand-made packages often used the label as the code).
func scaleLabelWeight(bulk pmsDomain.SKU, s pmsDomain.SKU) (int64, LinkSkipReason) {
	for _, code := range []string{s.Barcode, s.SKUCode} {
		decoded, err := pmsDomain.DecodeWeightEmbeddedBarcode(code)
		if err != nil {
			continue
		}
		if decoded.ScaleCode != *bulk.ScaleCode {
			return 0, LinkSkipOtherScaleCode
		}
		return decoded.WeightGrams, ""
	}
	return 0, LinkSkipNotAScaleLabel
}

func packageUpdate(bulk pmsDomain.SKU, p LinkedWeightPackage) UpdateSKUInput {
	in := UpdateSKUInputFrom(p.SKU)
	w := p.WeightGrams
	in.PortionedFromSKUID = &bulk.ID
	in.WeightGrams = &w
	in.UnitType = pmsDomain.SKUUnitTypeUnit
	in.PricingMode = pmsDomain.SKUPricingModeFixed
	in.PricePerKiloMinorUnits = nil
	in.PriceMinorUnits = p.PriceMinorUnits
	return in
}
