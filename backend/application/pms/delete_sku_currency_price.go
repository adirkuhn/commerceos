package pms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteSKUCurrencyPriceUseCase struct {
	db   *gorm.DB
	repo pmsDomain.SKUCurrencyPriceRepository
}

func NewDeleteSKUCurrencyPriceUseCase(db *gorm.DB, repo pmsDomain.SKUCurrencyPriceRepository) *DeleteSKUCurrencyPriceUseCase {
	return &DeleteSKUCurrencyPriceUseCase{db: db, repo: repo}
}

func (uc *DeleteSKUCurrencyPriceUseCase) Execute(ctx context.Context, skuID, currencyCode string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, skuID, currencyCode)
	})
	if err != nil {
		return fmt.Errorf("deleteSKUCurrencyPrice: %w", err)
	}
	return nil
}
