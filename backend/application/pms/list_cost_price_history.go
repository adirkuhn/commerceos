package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListCostPriceHistoryUseCase is the margin-trend-over-time chart's data
// source -- a thin read over SKUCostPriceHistoryRepository, most-recent
// first.
type ListCostPriceHistoryUseCase struct {
	db          *gorm.DB
	historyRepo pmsDomain.SKUCostPriceHistoryRepository
}

func NewListCostPriceHistoryUseCase(db *gorm.DB, historyRepo pmsDomain.SKUCostPriceHistoryRepository) *ListCostPriceHistoryUseCase {
	return &ListCostPriceHistoryUseCase{db: db, historyRepo: historyRepo}
}

func (uc *ListCostPriceHistoryUseCase) Execute(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUCostPriceHistory, error) {
	var result []pmsDomain.SKUCostPriceHistory
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.historyRepo.ListBySKU(ctx, skuID, limit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listCostPriceHistory: %w", err)
	}
	return result, nil
}
