package promotions

import (
	"context"
	"fmt"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeletePromotionUseCase deletes a promotion; its PromotionSKU lines cascade
// at the DB level (ON DELETE CASCADE). Any line still live on its SKU's
// price is reverted first -- see revertAppliedLines.
type DeletePromotionUseCase struct {
	db        *gorm.DB
	promoRepo promotionsDomain.PromotionRepository
	lineRepo  promotionsDomain.PromotionSKURepository
	reverter  lineReverter
}

func NewDeletePromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository, lineRepo promotionsDomain.PromotionSKURepository, reverter lineReverter) *DeletePromotionUseCase {
	return &DeletePromotionUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo, reverter: reverter}
}

func (uc *DeletePromotionUseCase) Execute(ctx context.Context, id string) error {
	var lines []promotionsDomain.PromotionSKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.promoRepo.FindByID(ctx, id); err != nil {
			return err
		}
		var err error
		lines, err = uc.lineRepo.ListByPromotion(ctx, id)
		return err
	})
	if err != nil {
		return fmt.Errorf("deletePromotion: %w", err)
	}
	if err := revertAppliedLines(ctx, uc.reverter, lines); err != nil {
		return fmt.Errorf("deletePromotion: %w", err)
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.promoRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deletePromotion: %w", err)
	}
	return nil
}
