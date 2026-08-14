package tax

import (
	"context"
	"fmt"
	"strings"
	"time"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpsertTaxRateInput struct {
	TaxClassID      string
	Country         string
	RateBasisPoints int32
}

type UpsertTaxRateUseCase struct {
	db        *gorm.DB
	rateRepo  taxDomain.TaxRateRepository
	classRepo taxDomain.TaxClassRepository
}

func NewUpsertTaxRateUseCase(db *gorm.DB, rateRepo taxDomain.TaxRateRepository, classRepo taxDomain.TaxClassRepository) *UpsertTaxRateUseCase {
	return &UpsertTaxRateUseCase{db: db, rateRepo: rateRepo, classRepo: classRepo}
}

func (uc *UpsertTaxRateUseCase) Execute(ctx context.Context, in UpsertTaxRateInput) (*taxDomain.TaxRate, error) {
	country := strings.ToUpper(strings.TrimSpace(in.Country))
	if len(country) != 2 {
		return nil, fmt.Errorf("upsertTaxRate: %w", taxDomain.ErrInvalidCountryCode)
	}
	if in.RateBasisPoints < 0 {
		return nil, fmt.Errorf("upsertTaxRate: %w", taxDomain.ErrInvalidTaxRateValue)
	}

	var rate taxDomain.TaxRate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.classRepo.FindByID(ctx, in.TaxClassID); err != nil {
			return err
		}
		now := time.Now()
		rate = taxDomain.TaxRate{
			TenantID: tenant.IDFromContext(ctx), TaxClassID: in.TaxClassID, Country: country,
			RateBasisPoints: in.RateBasisPoints, CreatedAt: now, UpdatedAt: now,
		}
		return uc.rateRepo.Upsert(ctx, &rate)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertTaxRate: %w", err)
	}
	return &rate, nil
}
