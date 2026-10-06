package promotions

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"strings"
	"time"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreatePromotionInput struct {
	Name     string
	Type     promotionsDomain.PromotionType
	StartsAt time.Time
	EndsAt   *time.Time
	// Status defaults to PromotionStatusDraft when empty.
	Status promotionsDomain.PromotionStatus
	Lines  []PromotionLineInput
}

// CreatePromotionResult is the created Promotion plus any non-blocking
// warnings surfaced during validation -- see validateLines' doc.
type CreatePromotionResult struct {
	Promotion promotionsDomain.Promotion
	Warnings  []string
}

type CreatePromotionUseCase struct {
	db         *gorm.DB
	promoRepo  promotionsDomain.PromotionRepository
	lineRepo   promotionsDomain.PromotionSKURepository
	skuChecker skuExistenceChecker
	batches    skuBatchLister
	outbox     eventrecord.Enqueuer
}

func NewCreatePromotionUseCase(db *gorm.DB, promoRepo promotionsDomain.PromotionRepository, lineRepo promotionsDomain.PromotionSKURepository, skuChecker skuExistenceChecker, batches skuBatchLister, outbox eventrecord.Enqueuer) *CreatePromotionUseCase {
	return &CreatePromotionUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo, skuChecker: skuChecker, batches: batches, outbox: outbox}
}

func (uc *CreatePromotionUseCase) Execute(ctx context.Context, in CreatePromotionInput) (*CreatePromotionResult, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("createPromotion: %w", promotionsDomain.ErrPromotionNameRequired)
	}
	if err := validatePromotionType(in.Type); err != nil {
		return nil, fmt.Errorf("createPromotion: %w", err)
	}
	if in.EndsAt != nil && !in.EndsAt.After(in.StartsAt) {
		return nil, fmt.Errorf("createPromotion: %w", promotionsDomain.ErrPromotionEndBeforeStart)
	}
	if in.Status == "" {
		in.Status = promotionsDomain.PromotionStatusDraft
	}
	if err := validatePromotionStatus(in.Status); err != nil {
		return nil, fmt.Errorf("createPromotion: %w", err)
	}

	if err := validateClearanceBatches(ctx, uc.batches, in.Lines, nil); err != nil {
		return nil, fmt.Errorf("createPromotion: %w", err)
	}

	var promo promotionsDomain.Promotion
	var warnings []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		warnings, err = validateLines(ctx, uc.skuChecker, in.Lines)
		if err != nil {
			return err
		}

		now := time.Now()
		promo = promotionsDomain.Promotion{
			TenantID: tenant.IDFromContext(ctx), Name: name, Type: in.Type,
			StartsAt: in.StartsAt, EndsAt: in.EndsAt, Status: in.Status,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.promoRepo.Create(ctx, &promo); err != nil {
			return err
		}

		created := make([]promotionsDomain.PromotionSKU, 0, len(in.Lines))
		for _, l := range in.Lines {
			line := promotionsDomain.PromotionSKU{
				TenantID: tenant.IDFromContext(ctx), PromotionID: promo.ID, SKUID: l.SKUID, BatchID: l.BatchID,
				DiscountType: l.DiscountType, DiscountValue: l.DiscountValue, CreatedAt: now,
			}
			if err := uc.lineRepo.Create(ctx, &line); err != nil {
				return err
			}
			created = append(created, line)
		}
		return recordPromotionCreated(ctx, uc.outbox, promo, created)
	})
	if err != nil {
		return nil, fmt.Errorf("createPromotion: %w", err)
	}
	return &CreatePromotionResult{Promotion: promo, Warnings: warnings}, nil
}
