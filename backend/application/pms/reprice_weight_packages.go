package pms

import (
	"context"
	"errors"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type RepriceWeightPackagesResult struct {
	Repriced int
}

// RepriceWeightPackagesUseCase sets every package of a weight pool to the
// pool's current package price (pmsDomain.PackagePriceMinorUnits) -- run
// when the pool's price per kilo changes (see
// events.SKUPricePerKiloChanged). Each package goes through the normal SKU
// update, so its price history is recorded and its channels are updated.
// Packages already at the right price are skipped, so a redelivered event
// changes nothing; a failed update is reported after trying the rest.
type RepriceWeightPackagesUseCase struct {
	db        *gorm.DB
	skus      skuReaderForPack
	variants  variantReaderForPack
	updateSKU skuUpdaterForLink
}

func NewRepriceWeightPackagesUseCase(db *gorm.DB, skus skuReaderForPack, variants variantReaderForPack, updateSKU skuUpdaterForLink) *RepriceWeightPackagesUseCase {
	return &RepriceWeightPackagesUseCase{db: db, skus: skus, variants: variants, updateSKU: updateSKU}
}

func (uc *RepriceWeightPackagesUseCase) Execute(ctx context.Context, poolSKUID string) (*RepriceWeightPackagesResult, error) {
	var pool pmsDomain.SKU
	var packages []pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skus.FindByID(ctx, poolSKUID)
		if err != nil {
			return err
		}
		pool = *sku
		if !isScaleSoldWeightPool(pool) {
			return nil
		}
		packages, err = uc.packagesOf(ctx, pool)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("repriceWeightPackages: %w", err)
	}

	result := &RepriceWeightPackagesResult{}
	var failures []error
	for _, p := range packages {
		price := pmsDomain.PackagePriceMinorUnits(*pool.PricePerKiloMinorUnits, *p.WeightGrams)
		if price == p.PriceMinorUnits {
			continue
		}
		in := UpdateSKUInputFrom(p)
		in.PriceMinorUnits = price
		if _, err := uc.updateSKU.Execute(ctx, in); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", p.SKUCode, err))
			continue
		}
		result.Repriced++
	}
	if len(failures) > 0 {
		return result, fmt.Errorf("repriceWeightPackages: %w", errors.Join(failures...))
	}
	return result, nil
}

// packagesOf lists the SKUs portioned off pool across its product's
// variants, with a weight to price them by.
func (uc *RepriceWeightPackagesUseCase) packagesOf(ctx context.Context, pool pmsDomain.SKU) ([]pmsDomain.SKU, error) {
	poolVariant, err := uc.variants.FindByID(ctx, pool.VariantID)
	if err != nil {
		return nil, err
	}
	variants, err := uc.variants.ListByProduct(ctx, poolVariant.ProductID)
	if err != nil {
		return nil, err
	}
	var out []pmsDomain.SKU
	for _, v := range variants {
		skus, err := uc.skus.ListByVariant(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range skus {
			if s.PortionedFromSKUID != nil && *s.PortionedFromSKUID == pool.ID && s.WeightGrams != nil {
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// Reprice is Execute without the result -- the shape an event consumer
// needs (see infrastructure/consumers.SKUPricePerKiloChangedConsumer).
func (uc *RepriceWeightPackagesUseCase) Reprice(ctx context.Context, poolSKUID string) error {
	_, err := uc.Execute(ctx, poolSKUID)
	return err
}
