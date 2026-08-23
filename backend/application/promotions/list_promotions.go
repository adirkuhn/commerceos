package promotions

import (
	"context"
	"fmt"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListPromotionsUseCase struct {
	db        *gorm.DB
	promoRepo promotionsDomain.PromotionRepository
}

func NewListPromotionsUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository) *ListPromotionsUseCase {
	return &ListPromotionsUseCase{db: db, promoRepo: promoRepo}
}

func (uc *ListPromotionsUseCase) Execute(ctx context.Context) ([]promotionsDomain.Promotion, error) {
	var result []promotionsDomain.Promotion
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.promoRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listPromotions: %w", err)
	}
	return result, nil
}
