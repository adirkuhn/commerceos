package promotions

import (
	"context"
	"fmt"
	"strings"
	"time"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdatePromotionInput struct {
	ID       string
	Name     string
	Type     promotionsDomain.PromotionType
	StartsAt time.Time
	EndsAt   *time.Time
	Status   promotionsDomain.PromotionStatus
	Lines    []PromotionLineInput
}

type UpdatePromotionUseCase struct {
	db         *gorm.DB
	promoRepo  promotionsDomain.PromotionRepository
	lineRepo   promotionsDomain.PromotionSKURepository
	skuChecker skuExistenceChecker
	batches    skuBatchLister
	reverter   lineReverter
}

func NewUpdatePromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository, lineRepo promotionsDomain.PromotionSKURepository, skuChecker skuExistenceChecker, batches skuBatchLister, reverter lineReverter) *UpdatePromotionUseCase {
	return &UpdatePromotionUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo, skuChecker: skuChecker, batches: batches, reverter: reverter}
}

// Execute replaces the promotion's whole line set wholesale (delete then
// recreate) rather than diffing individual lines -- simplest correct
// approach for a small per-promotion line count; same tradeoff UpdateSKU's
// sibling domains accept elsewhere for small child collections.
//
// Three steps, because reverting a price opens its own transaction:
// validate everything (read-only), restore the price of any line currently
// live (see revertAppliedLines), then write. The new lines start
// un-applied; the promotion_activation sweep applies them within a minute
// with a fresh snapshot of the SKU's real pre-promotion price.
func (uc *UpdatePromotionUseCase) Execute(ctx context.Context, in UpdatePromotionInput) (*promotionsDomain.Promotion, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("updatePromotion: %w", promotionsDomain.ErrPromotionNameRequired)
	}
	if err := validatePromotionType(in.Type); err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}
	if in.EndsAt != nil && !in.EndsAt.After(in.StartsAt) {
		return nil, fmt.Errorf("updatePromotion: %w", promotionsDomain.ErrPromotionEndBeforeStart)
	}
	if err := validatePromotionStatus(in.Status); err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}

	var existingLines []promotionsDomain.PromotionSKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.promoRepo.FindByID(ctx, in.ID); err != nil {
			return err
		}
		if _, err := validateLines(ctx, uc.skuChecker, in.Lines); err != nil {
			return err
		}
		var err error
		existingLines, err = uc.lineRepo.ListByPromotion(ctx, in.ID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}
	if err := validateClearanceBatches(ctx, uc.batches, in.Lines, clearanceBatchIDs(existingLines)); err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}
	if err := revertAppliedLines(ctx, uc.reverter, existingLines); err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}

	var promo promotionsDomain.Promotion
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.promoRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}

		existing.Name = name
		existing.Type = in.Type
		existing.StartsAt = in.StartsAt
		existing.EndsAt = in.EndsAt
		existing.Status = in.Status
		existing.UpdatedAt = time.Now()
		if err := uc.promoRepo.Update(ctx, existing); err != nil {
			return err
		}
		promo = *existing

		if err := uc.lineRepo.DeleteByPromotion(ctx, promo.ID); err != nil {
			return err
		}
		now := time.Now()
		for _, l := range in.Lines {
			line := promotionsDomain.PromotionSKU{
				TenantID: tenant.IDFromContext(ctx), PromotionID: promo.ID, SKUID: l.SKUID, BatchID: l.BatchID,
				DiscountType: l.DiscountType, DiscountValue: l.DiscountValue, CreatedAt: now,
			}
			if err := uc.lineRepo.Create(ctx, &line); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}
	return &promo, nil
}
