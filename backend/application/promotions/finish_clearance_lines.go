package promotions

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"time"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// lineReverter narrows RevertPromotionSKUUseCase to what this use case
// needs (ISP).
type lineReverter interface {
	Execute(ctx context.Context, line promotionsDomain.PromotionSKU) error
}

// FinishClearanceLinesUseCase ends clearance lines whose batch is finished
// (see domain/promotions.ClearanceBatch.IsFinished): the line is marked
// finished, its SKU goes back to the sale price it had before the promotion
// (usually none, i.e. the regular price), and the promotion itself is ended
// once none of its lines are left unfinished. Driven both by the
// inventory.stock.changed consumer (sales, discards, write-offs -- reacts
// within seconds) and by the promotion_activation sweep (expiry dates
// passing and batch deletes, which raise no stock event).
type FinishClearanceLinesUseCase struct {
	db        *gorm.DB
	batches   skuBatchLister
	lines     promotionsDomain.ClearanceLineRepository
	promoRepo promotionsDomain.PromotionRepository
	reverter  lineReverter
	outbox    eventrecord.Enqueuer
}

func NewFinishClearanceLinesUseCase(db *gorm.DB, batches skuBatchLister, lines promotionsDomain.ClearanceLineRepository, promoRepo promotionsDomain.PromotionRepository, reverter lineReverter, outbox eventrecord.Enqueuer) *FinishClearanceLinesUseCase {
	return &FinishClearanceLinesUseCase{db: db, batches: batches, lines: lines, promoRepo: promoRepo, reverter: reverter, outbox: outbox}
}

// ExecuteForSKU checks every open clearance line on skuID for ctx's tenant.
// One line's failure doesn't stop the others; all errors are returned.
func (uc *FinishClearanceLinesUseCase) ExecuteForSKU(ctx context.Context, skuID string) error {
	var open []promotionsDomain.PromotionSKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		open, err = uc.lines.ListOpenBySKU(ctx, skuID)
		return err
	})
	if err != nil {
		return fmt.Errorf("finishClearanceLinesForSKU: %w", err)
	}
	var errs []error
	for _, line := range open {
		if _, err := uc.ExecuteForLine(ctx, line); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ExecuteForLine finishes line if its batch is done, reporting whether it
// did. The finish is committed before the price revert: if the revert then
// fails, the line is still applied+finished, which ListDueForReversal picks
// up on the next sweep -- the price is never left discounted for good.
func (uc *FinishClearanceLinesUseCase) ExecuteForLine(ctx context.Context, line promotionsDomain.PromotionSKU) (bool, error) {
	if line.BatchID == nil {
		return false, nil
	}
	batches, err := uc.batches.Execute(ctx, line.SKUID)
	if err != nil {
		return false, fmt.Errorf("finishClearanceLine %s: %w", line.ID, err)
	}
	now := time.Now()
	if !clearanceBatchFor(batches, *line.BatchID).IsFinished(now) {
		return false, nil
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.lines.MarkFinished(ctx, line.ID, now); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.PromotionLineFinished, line.ID, eventrecord.Data{
			"promotion_id": line.PromotionID, "sku_id": line.SKUID, "batch_id": *line.BatchID,
		}); err != nil {
			return err
		}
		return uc.endPromotionIfDone(ctx, line.PromotionID, now)
	})
	if err != nil {
		return false, fmt.Errorf("finishClearanceLine %s: %w", line.ID, err)
	}

	if line.Applied {
		if err := uc.reverter.Execute(ctx, line); err != nil {
			return true, fmt.Errorf("finishClearanceLine %s: %w", line.ID, err)
		}
	}
	return true, nil
}

func (uc *FinishClearanceLinesUseCase) endPromotionIfDone(ctx context.Context, promotionID string, now time.Time) error {
	unfinished, err := uc.lines.CountUnfinished(ctx, promotionID)
	if err != nil || unfinished > 0 {
		return err
	}
	promo, err := uc.promoRepo.FindByID(ctx, promotionID)
	if err != nil {
		return err
	}
	if promo.Status == promotionsDomain.PromotionStatusEnded {
		return nil
	}
	before := *promo
	promo.Status = promotionsDomain.PromotionStatusEnded
	promo.UpdatedAt = now
	if err := uc.promoRepo.Update(ctx, promo); err != nil {
		return err
	}
	return eventrecord.Record(ctx, uc.outbox, events.PromotionUpdated, promo.ID,
		eventrecord.NewChanges().Set("status", string(before.Status), string(promo.Status)),
		eventrecord.Data{"reason": PromotionChangeReasonClearanceFinished})
}
