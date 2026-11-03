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
//
// Before recalculating a SKU it fills in the cost of the unit a case SKU is
// sold out of (see pmsapp.SyncUnitCostFromCaseUseCase) -- the catch-all for
// case links whose unit never got a cost (links made before that existed,
// or an event that never arrived).
type RunCostPriceRecalculationUseCase struct {
	db                      *gorm.DB
	skuRepo                 activeSKULister
	recalculateCostPricesUC costPriceRecalculator
	unitCostSyncer          unitCostFromCaseSyncer
	historySyncer           costPriceFromHistorySyncer
}

// costPriceFromHistorySyncer is pmsapp.SyncCostPriceFromHistoryUseCase (ISP).
type costPriceFromHistorySyncer interface {
	Execute(ctx context.Context, skuID string) error
}

type activeSKULister interface {
	ListActive(ctx context.Context) ([]pmsDomain.SKU, error)
}

// costPriceRecalculator is pmsapp.RecalculateCostPricesUseCase (ISP).
type costPriceRecalculator interface {
	Execute(ctx context.Context, skuID string) (*pmsapp.RecalculateCostPricesResult, error)
}

// unitCostFromCaseSyncer is pmsapp.SyncUnitCostFromCaseUseCase (ISP).
type unitCostFromCaseSyncer interface {
	SyncFromCase(ctx context.Context, skuID string) error
}

func NewRunCostPriceRecalculationUseCase(db *gorm.DB, skuRepo activeSKULister, recalculateCostPricesUC costPriceRecalculator,
	unitCostSyncer unitCostFromCaseSyncer, historySyncer costPriceFromHistorySyncer) *RunCostPriceRecalculationUseCase {
	return &RunCostPriceRecalculationUseCase{db: db, skuRepo: skuRepo, recalculateCostPricesUC: recalculateCostPricesUC,
		unitCostSyncer: unitCostSyncer, historySyncer: historySyncer}
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
		if err := uc.historySyncer.Execute(runCtx, s.ID); err != nil {
			log.Printf("sync cost price from supplier history sku %s: %v", s.ID, err)
		}
		if err := uc.unitCostSyncer.SyncFromCase(runCtx, s.ID); err != nil {
			log.Printf("sync unit cost from case sku %s: %v", s.ID, err)
		}
		if _, err := uc.recalculateCostPricesUC.Execute(runCtx, s.ID); err != nil {
			log.Printf("recalculate cost price for sku %s: %v", s.ID, err)
		}
	}
	return nil
}
