package promotions

import (
	"context"
	"fmt"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetPromotionResult struct {
	Promotion promotionsDomain.Promotion
	Lines     []promotionsDomain.PromotionSKU
}

type GetPromotionUseCase struct {
	db        *gorm.DB
	promoRepo promotionsDomain.PromotionRepository
	lineRepo  promotionsDomain.PromotionSKURepository
}

func NewGetPromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository, lineRepo promotionsDomain.PromotionSKURepository) *GetPromotionUseCase {
	return &GetPromotionUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo}
}

func (uc *GetPromotionUseCase) Execute(ctx context.Context, id string) (*GetPromotionResult, error) {
	var result GetPromotionResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		promo, err := uc.promoRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		lines, err := uc.lineRepo.ListByPromotion(ctx, id)
		if err != nil {
			return err
		}
		result = GetPromotionResult{Promotion: *promo, Lines: lines}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPromotion: %w", err)
	}
	return &result, nil
}
