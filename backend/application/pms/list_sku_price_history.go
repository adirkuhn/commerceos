package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListSKUPriceHistoryUseCase is the selling-price audit trail's read side --
// a thin read over SKUPriceHistoryRepository, most recent interval first.
type ListSKUPriceHistoryUseCase struct {
	db          *gorm.DB
	historyRepo pmsDomain.SKUPriceHistoryRepository
}

func NewListSKUPriceHistoryUseCase(db *gorm.DB, historyRepo pmsDomain.SKUPriceHistoryRepository) *ListSKUPriceHistoryUseCase {
	return &ListSKUPriceHistoryUseCase{db: db, historyRepo: historyRepo}
}

func (uc *ListSKUPriceHistoryUseCase) Execute(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUPriceHistory, error) {
	var result []pmsDomain.SKUPriceHistory
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.historyRepo.ListBySKU(ctx, skuID, limit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSKUPriceHistory: %w", err)
	}
	return result, nil
}
