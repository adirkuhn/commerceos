package pms

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UpsertSKUCurrencyPriceInput struct {
	SKUID               string
	CurrencyCode        string
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	SaleEndDate         *time.Time
}

type UpsertSKUCurrencyPriceUseCase struct {
	db      *gorm.DB
	skuRepo pmsDomain.SKURepository
	repo    pmsDomain.SKUCurrencyPriceRepository
}

func NewUpsertSKUCurrencyPriceUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, repo pmsDomain.SKUCurrencyPriceRepository) *UpsertSKUCurrencyPriceUseCase {
	return &UpsertSKUCurrencyPriceUseCase{db: db, skuRepo: skuRepo, repo: repo}
}

func (uc *UpsertSKUCurrencyPriceUseCase) Execute(ctx context.Context, in UpsertSKUCurrencyPriceInput) (*pmsDomain.SKUCurrencyPrice, error) {
	if !fxDomain.ValidCurrencyCode(in.CurrencyCode) {
		return nil, fmt.Errorf("upsertSKUCurrencyPrice: %w", fxDomain.ErrInvalidCurrencyCode)
	}
	if in.SalePriceMinorUnits != nil && *in.SalePriceMinorUnits > in.PriceMinorUnits {
		return nil, fmt.Errorf("upsertSKUCurrencyPrice: %w", pmsDomain.ErrSalePriceExceedsPrice)
	}
	if in.SaleEndDate != nil && in.SalePriceMinorUnits == nil {
		return nil, fmt.Errorf("upsertSKUCurrencyPrice: %w", pmsDomain.ErrSaleEndDateRequiresSalePrice)
	}

	var price pmsDomain.SKUCurrencyPrice
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}
		now := time.Now()
		price = pmsDomain.SKUCurrencyPrice{
			TenantID: tenant.IDFromContext(ctx), SKUID: sku.ID, CurrencyCode: in.CurrencyCode,
			PriceMinorUnits: in.PriceMinorUnits, SalePriceMinorUnits: in.SalePriceMinorUnits, SaleEndDate: in.SaleEndDate,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.repo.Upsert(ctx, &price)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertSKUCurrencyPrice: %w", err)
	}
	return &price, nil
}
