package pms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListSKUCurrencyPricesUseCase struct {
	db   *gorm.DB
	repo pmsDomain.SKUCurrencyPriceRepository
}

func NewListSKUCurrencyPricesUseCase(db *gorm.DB, repo pmsDomain.SKUCurrencyPriceRepository) *ListSKUCurrencyPricesUseCase {
	return &ListSKUCurrencyPricesUseCase{db: db, repo: repo}
}

func (uc *ListSKUCurrencyPricesUseCase) Execute(ctx context.Context, skuID string) ([]pmsDomain.SKUCurrencyPrice, error) {
	var prices []pmsDomain.SKUCurrencyPrice
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.repo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		prices = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listSKUCurrencyPrices: %w", err)
	}
	return prices, nil
}
