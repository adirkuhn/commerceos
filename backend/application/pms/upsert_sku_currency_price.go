package pms

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	outbox  eventrecord.Enqueuer
}

func NewUpsertSKUCurrencyPriceUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, repo pmsDomain.SKUCurrencyPriceRepository, outbox eventrecord.Enqueuer) *UpsertSKUCurrencyPriceUseCase {
	return &UpsertSKUCurrencyPriceUseCase{db: db, skuRepo: skuRepo, repo: repo, outbox: outbox}
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
		previous, err := uc.repo.FindBySKUAndCurrency(ctx, sku.ID, in.CurrencyCode)
		if err != nil {
			return err
		}
		var before pmsDomain.SKUCurrencyPrice
		if previous != nil {
			before = *previous
		}
		now := time.Now()
		price = pmsDomain.SKUCurrencyPrice{
			TenantID: tenant.IDFromContext(ctx), SKUID: sku.ID, CurrencyCode: in.CurrencyCode,
			PriceMinorUnits: in.PriceMinorUnits, SalePriceMinorUnits: in.SalePriceMinorUnits, SaleEndDate: in.SaleEndDate,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Upsert(ctx, &price); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SKUCurrencyPriceSaved, sku.ID,
			eventrecord.Data{"currency_code": price.CurrencyCode}, eventrecord.NewChanges().
				Set("price_minor_units", before.PriceMinorUnits, price.PriceMinorUnits).
				Set("sale_price_minor_units", before.SalePriceMinorUnits, price.SalePriceMinorUnits).
				Set("sale_end_date", before.SaleEndDate, price.SaleEndDate))
	})
	if err != nil {
		return nil, fmt.Errorf("upsertSKUCurrencyPrice: %w", err)
	}
	return &price, nil
}
