package promotions

import (
	"context"
	"errors"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PromotionLineView is a PromotionSKU line plus its SKU's current code --
// a UI showing a bare sku_id UUID is unusable to staff, so Get enriches
// each line at read time rather than returning the raw domain rows.
// SKUCode is "" when the SKU has since been deleted.
type PromotionLineView struct {
	promotionsDomain.PromotionSKU
	SKUCode string
}

type GetPromotionResult struct {
	Promotion promotionsDomain.Promotion
	Lines     []PromotionLineView
}

type GetPromotionUseCase struct {
	db         *gorm.DB
	promoRepo  promotionsDomain.PromotionRepository
	lineRepo   promotionsDomain.PromotionSKURepository
	skuChecker skuExistenceChecker
}

func NewGetPromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository, lineRepo promotionsDomain.PromotionSKURepository, skuChecker skuExistenceChecker) *GetPromotionUseCase {
	return &GetPromotionUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo, skuChecker: skuChecker}
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
		views := make([]PromotionLineView, len(lines))
		for i, line := range lines {
			view := PromotionLineView{PromotionSKU: line}
			sku, err := uc.skuChecker.FindByID(ctx, line.SKUID)
			if err != nil && !errors.Is(err, pmsDomain.ErrSKUNotFound) {
				return err
			}
			if sku != nil {
				view.SKUCode = sku.SKUCode
			}
			views[i] = view
		}
		result = GetPromotionResult{Promotion: *promo, Lines: views}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPromotion: %w", err)
	}
	return &result, nil
}
