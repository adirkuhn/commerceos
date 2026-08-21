package scheduler

import (
	"context"
	"log"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RunExpiredBatchesUseCase flips Batch.Status to expired for every batch
// past its tenant's configured grace period, across every tenant -- moved
// out of cmd/scheduler/main.go's own runExpiredBatches so both the periodic
// poller and a super_admin's "run now" share one implementation. A failure
// on one batch is logged, not fatal -- the next run tries again.
type RunExpiredBatchesUseCase struct {
	db            *gorm.DB
	batchRepo     invDomain.BatchRepository
	expireBatchUC *invapp.ExpireBatchUseCase
}

func NewRunExpiredBatchesUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, expireBatchUC *invapp.ExpireBatchUseCase) *RunExpiredBatchesUseCase {
	return &RunExpiredBatchesUseCase{db: db, batchRepo: batchRepo, expireBatchUC: expireBatchUC}
}

func (uc *RunExpiredBatchesUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var toExpire []invDomain.BatchToExpire
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		toExpire, err = uc.batchRepo.ListToExpire(ctx)
		return err
	})
	if err != nil {
		return err
	}

	for _, b := range toExpire {
		runCtx := tenant.WithTenantID(adminCtx, b.TenantID)
		if err := uc.expireBatchUC.Execute(runCtx, b.BatchID); err != nil {
			log.Printf("expire batch %s: %v", b.BatchID, err)
		}
	}
	return nil
}
