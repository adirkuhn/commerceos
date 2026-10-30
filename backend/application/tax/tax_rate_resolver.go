package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

// TaxRateResolver answers "which VAT rate applies to this tax class in this
// country" for other modules. It reads inside the caller's transaction
// (tenant.WithTenant), so it opens none of its own.
type TaxRateResolver struct {
	classRepo taxDomain.TaxClassRepository
	rateRepo  taxDomain.TaxRateRepository
}

func NewTaxRateResolver(classRepo taxDomain.TaxClassRepository, rateRepo taxDomain.TaxRateRepository) *TaxRateResolver {
	return &TaxRateResolver{classRepo: classRepo, rateRepo: rateRepo}
}

// RateBasisPoints is taxClassID's rate in country (nil = the default
// class); 0 when none is configured, the same "unset is zero" reading as
// checkout's tax.
func (r *TaxRateResolver) RateBasisPoints(ctx context.Context, taxClassID *string, country string) (int64, error) {
	bp, _, err := r.Rate(ctx, taxClassID, country)
	return bp, err
}

// Rate is RateBasisPoints plus whether a rate is configured at all: false
// when there is no default class (for a nil taxClassID) or the class has no
// rate for country -- a configured 0% rate is configured.
func (r *TaxRateResolver) Rate(ctx context.Context, taxClassID *string, country string) (int64, bool, error) {
	classes, err := r.classRepo.List(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("taxRateResolver: %w", err)
	}
	rates, err := r.rateRepo.List(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("taxRateResolver: %w", err)
	}
	bp, ok := taxDomain.RateFor(classes, rates, taxClassID, country)
	return int64(bp), ok, nil
}
