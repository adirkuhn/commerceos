package promotions

import (
	"context"
	"fmt"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeletePromotionUseCase deletes a promotion; its PromotionSKU lines cascade
// at the DB level (ON DELETE CASCADE).
type DeletePromotionUseCase struct {
	db        *gorm.DB
	promoRepo promotionsDomain.PromotionRepository
}

func NewDeletePromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository) *DeletePromotionUseCase {
	return &DeletePromotionUseCase{db: db, promoRepo: promoRepo}
}

func (uc *DeletePromotionUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.promoRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.promoRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deletePromotion: %w", err)
	}
	return nil
}
