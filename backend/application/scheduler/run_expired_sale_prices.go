package scheduler

import (
	"context"
	"log"

	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RunExpiredSalePricesUseCase clears SalePriceMinorUnits on every SKU whose
// SaleEndDate has passed, across every tenant -- moved out of
// cmd/scheduler/main.go's own runExpiredSalePrices so both the periodic
// poller and a super_admin's "run now" (application/scheduler.RunJobNowUseCase)
// share one implementation. A failure on one SKU is logged, not fatal -- the
// next run tries again; only a failure listing the candidates at all fails
// the whole run.
type RunExpiredSalePricesUseCase struct {
	db                *gorm.DB
	skuRepo           pmsDomain.SKURepository
	expireSalePriceUC *pmsapp.ExpireSKUSalePriceUseCase
}

func NewRunExpiredSalePricesUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, expireSalePriceUC *pmsapp.ExpireSKUSalePriceUseCase) *RunExpiredSalePricesUseCase {
	return &RunExpiredSalePricesUseCase{db: db, skuRepo: skuRepo, expireSalePriceUC: expireSalePriceUC}
}

func (uc *RunExpiredSalePricesUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var expired []pmsDomain.SKU
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		expired, err = uc.skuRepo.ListWithExpiredSale(ctx)
		return err
	})
	if err != nil {
		return err
	}

	for _, s := range expired {
		runCtx := tenant.WithTenantID(adminCtx, s.TenantID)
		if _, err := uc.expireSalePriceUC.Execute(runCtx, s.ID); err != nil {
			log.Printf("expire sale price for sku %s: %v", s.ID, err)
		}
	}
	return nil
}
