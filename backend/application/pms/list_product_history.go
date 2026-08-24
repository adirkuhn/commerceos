package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListProductHistoryUseCase is the product/variant name+status audit
// trail's read side -- a thin read over ProductHistoryRepository, most
// recent change first.
type ListProductHistoryUseCase struct {
	db          *gorm.DB
	historyRepo pmsDomain.ProductHistoryRepository
}

func NewListProductHistoryUseCase(db *gorm.DB, historyRepo pmsDomain.ProductHistoryRepository) *ListProductHistoryUseCase {
	return &ListProductHistoryUseCase{db: db, historyRepo: historyRepo}
}

func (uc *ListProductHistoryUseCase) Execute(ctx context.Context, entityType pmsDomain.ProductHistoryEntityType, entityID string) ([]pmsDomain.ProductHistory, error) {
	var result []pmsDomain.ProductHistory
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.historyRepo.ListByEntity(ctx, entityType, entityID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listProductHistory: %w", err)
	}
	return result, nil
}
