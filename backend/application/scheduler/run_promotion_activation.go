package scheduler

import (
	"context"
	"log"
	"time"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// openClearanceLines narrows promotionsDomain.ClearanceLineRepository to the
// one cross-tenant read this sweep needs (ISP).
type openClearanceLines interface {
	ListOpen(ctx context.Context) ([]promotionsDomain.PromotionSKU, error)
}

// clearanceLineFinisher narrows apppromotions.FinishClearanceLinesUseCase
// (ISP).
type clearanceLineFinisher interface {
	ExecuteForLine(ctx context.Context, line promotionsDomain.PromotionSKU) (bool, error)
}

// RunPromotionActivationUseCase activates newly-active promotions'
// SalePriceMinorUnits and reverts newly-ended ones, across every tenant --
// see domain/promotions.PromotionSKU's doc for what "activate"/"revert"
// mean. It first ends any clearance line whose batch is finished: the
// inventory.stock.changed consumer handles sales and discards as they
// happen, but an expiry date passing or a batch being deleted raises no
// event, so this sweep is what catches those. Same log-and-continue-on-one-
// failure tolerance as RunExpiredSalePricesUseCase/RunExpiredBatchesUseCase.
type RunPromotionActivationUseCase struct {
	db             *gorm.DB
	lineRepo       promotionsDomain.PromotionSKURepository
	applyUC        *apppromotions.ApplyPromotionSKUUseCase
	revertUC       *apppromotions.RevertPromotionSKUUseCase
	clearanceLines openClearanceLines
	finishUC       clearanceLineFinisher
}

func NewRunPromotionActivationUseCase(db *gorm.DB, lineRepo promotionsDomain.PromotionSKURepository, applyUC *apppromotions.ApplyPromotionSKUUseCase, revertUC *apppromotions.RevertPromotionSKUUseCase, clearanceLines openClearanceLines, finishUC clearanceLineFinisher) *RunPromotionActivationUseCase {
	return &RunPromotionActivationUseCase{db: db, lineRepo: lineRepo, applyUC: applyUC, revertUC: revertUC, clearanceLines: clearanceLines, finishUC: finishUC}
}

func (uc *RunPromotionActivationUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)
	now := time.Now()

	if err := uc.finishClearanceLines(adminCtx); err != nil {
		return err
	}

	var toActivate []promotionsDomain.PromotionSKU
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		toActivate, err = uc.lineRepo.ListDueForActivation(ctx, now)
		return err
	})
	if err != nil {
		return err
	}
	for _, line := range toActivate {
		runCtx := tenant.WithTenantID(adminCtx, line.TenantID)
		if err := uc.applyUC.Execute(runCtx, line); err != nil {
			log.Printf("activate promotion sku line %s: %v", line.ID, err)
		}
	}

	var toRevert []promotionsDomain.PromotionSKU
	err = tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		toRevert, err = uc.lineRepo.ListDueForReversal(ctx, now)
		return err
	})
	if err != nil {
		return err
	}
	for _, line := range toRevert {
		runCtx := tenant.WithTenantID(adminCtx, line.TenantID)
		if err := uc.revertUC.Execute(runCtx, line); err != nil {
			log.Printf("revert promotion sku line %s: %v", line.ID, err)
		}
	}
	return nil
}

// finishClearanceLines runs before activation so a clearance whose batch is
// already done is never applied in the first place.
func (uc *RunPromotionActivationUseCase) finishClearanceLines(adminCtx context.Context) error {
	var open []promotionsDomain.PromotionSKU
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		open, err = uc.clearanceLines.ListOpen(ctx)
		return err
	})
	if err != nil {
		return err
	}
	for _, line := range open {
		runCtx := tenant.WithTenantID(adminCtx, line.TenantID)
		if _, err := uc.finishUC.ExecuteForLine(runCtx, line); err != nil {
			log.Printf("finish clearance promotion line %s: %v", line.ID, err)
		}
	}
	return nil
}
