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
}

func NewUpdatePromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository, lineRepo promotionsDomain.PromotionSKURepository, skuChecker skuExistenceChecker, batches skuBatchLister) *UpdatePromotionUseCase {
	return &UpdatePromotionUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo, skuChecker: skuChecker, batches: batches}
}

// Execute replaces the promotion's whole line set wholesale (delete then
// recreate) rather than diffing individual lines -- simplest correct
// approach for a small per-promotion line count; same tradeoff UpdateSKU's
// sibling domains accept elsewhere for small child collections.
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

	if err := uc.validateClearanceBatches(ctx, in); err != nil {
		return nil, fmt.Errorf("updatePromotion: %w", err)
	}

	var promo promotionsDomain.Promotion
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.promoRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if _, err := validateLines(ctx, uc.skuChecker, in.Lines); err != nil {
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

// validateClearanceBatches validates only clearance batches new to this
// promotion -- see the package-level validateClearanceBatches. Skips the
// extra read entirely when no line is a clearance line.
func (uc *UpdatePromotionUseCase) validateClearanceBatches(ctx context.Context, in UpdatePromotionInput) error {
	if !hasClearanceLine(in.Lines) {
		return nil
	}
	existing := map[string]bool{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		lines, err := uc.lineRepo.ListByPromotion(ctx, in.ID)
		if err != nil {
			return err
		}
		for _, l := range lines {
			if l.BatchID != nil {
				existing[*l.BatchID] = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return validateClearanceBatches(ctx, uc.batches, in.Lines, existing)
}
