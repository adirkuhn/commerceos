package scheduler

import (
	"context"
	"log"

	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RunCostPriceRecalculationUseCase recomputes every active SKU's
// weighted-average cost price and recommended price, across every tenant --
// moved out of cmd/scheduler/main.go's own runRecalculateCostPrices so both
// the periodic poller and a super_admin's "run now" share one
// implementation. Dropped that function's process-local once-per-UTC-day
// gate: the persisted scheduled_jobs row's NextRunAt/IntervalMinutes is now
// what gates this (see application/scheduler.RunJobNowUseCase and
// cmd/scheduler's due-check loop), the same mechanism SyncSchedule already
// uses. A failure on one SKU is logged, not fatal.
type RunCostPriceRecalculationUseCase struct {
	db                      *gorm.DB
	skuRepo                 pmsDomain.SKURepository
	recalculateCostPricesUC *pmsapp.RecalculateCostPricesUseCase
}

func NewRunCostPriceRecalculationUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, recalculateCostPricesUC *pmsapp.RecalculateCostPricesUseCase) *RunCostPriceRecalculationUseCase {
	return &RunCostPriceRecalculationUseCase{db: db, skuRepo: skuRepo, recalculateCostPricesUC: recalculateCostPricesUC}
}

func (uc *RunCostPriceRecalculationUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var skus []pmsDomain.SKU
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		skus, err = uc.skuRepo.ListActive(ctx)
		return err
	})
	if err != nil {
		return err
	}

	for _, s := range skus {
		runCtx := tenant.WithTenantID(adminCtx, s.TenantID)
		if _, err := uc.recalculateCostPricesUC.Execute(runCtx, s.ID); err != nil {
			log.Printf("recalculate cost price for sku %s: %v", s.ID, err)
		}
	}
	return nil
}
