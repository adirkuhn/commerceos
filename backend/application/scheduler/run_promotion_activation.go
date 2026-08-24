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

// RunPromotionActivationUseCase activates newly-active promotions'
// SalePriceMinorUnits and reverts newly-ended ones, across every tenant --
// see domain/promotions.PromotionSKU's doc for what "activate"/"revert"
// mean. Same log-and-continue-on-one-failure tolerance as
// RunExpiredSalePricesUseCase/RunExpiredBatchesUseCase.
type RunPromotionActivationUseCase struct {
	db       *gorm.DB
	lineRepo promotionsDomain.PromotionSKURepository
	applyUC  *apppromotions.ApplyPromotionSKUUseCase
	revertUC *apppromotions.RevertPromotionSKUUseCase
}

func NewRunPromotionActivationUseCase(db *gorm.DB, lineRepo promotionsDomain.PromotionSKURepository, applyUC *apppromotions.ApplyPromotionSKUUseCase, revertUC *apppromotions.RevertPromotionSKUUseCase) *RunPromotionActivationUseCase {
	return &RunPromotionActivationUseCase{db: db, lineRepo: lineRepo, applyUC: applyUC, revertUC: revertUC}
}

func (uc *RunPromotionActivationUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)
	now := time.Now()

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
