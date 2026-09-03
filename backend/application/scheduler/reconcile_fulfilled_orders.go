package scheduler

import (
	"context"
	"log"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// reconcileFulfilledOrdersLimit bounds one run's work -- same "small,
// bounded sweep" convention as application/shipping.reconcileBatchLimit;
// this runs on a 24h scheduler tick, not a hot path.
const reconcileFulfilledOrdersLimit = 500

// ReconcileFulfilledOrdersUseCase is the 24h self-heal sweep for orders
// whose shipment already fulfilled them (SalesOrder.Fulfilled = true) but
// whose Status never made it to Shipped/OnHold -- both webhook paths that
// normally drive that transition (HandleShipmentWebhookUseCase,
// HandleOrderStatusWebhookUseCase) can be missed the same way any webhook
// can. Reuses appshipping.MarkOrderShippedOnFulfillment so this sweep can
// never decide the Shipped-vs-OnHold-for-Woo question any differently than
// the webhook paths do. A failure on one order is logged, not fatal -- the
// next run tries again; only a failure listing the candidates at all fails
// the whole run.
type ReconcileFulfilledOrdersUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	updater        appshipping.SalesOrderStatusUpdater
}

func NewReconcileFulfilledOrdersUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, updater appshipping.SalesOrderStatusUpdater) *ReconcileFulfilledOrdersUseCase {
	return &ReconcileFulfilledOrdersUseCase{db: db, salesOrderRepo: salesOrderRepo, updater: updater}
}

func (uc *ReconcileFulfilledOrdersUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	fulfilled := true
	var stuck []omsDomain.SalesOrder
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		stuck, _, err = uc.salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
			Fulfilled: &fulfilled,
			Status:    []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing},
			Limit:     reconcileFulfilledOrdersLimit,
		})
		return err
	})
	if err != nil {
		return err
	}

	for _, o := range stuck {
		runCtx := tenant.WithTenantID(adminCtx, o.TenantID)
		if err := appshipping.MarkOrderShippedOnFulfillment(runCtx, uc.db, uc.salesOrderRepo, uc.updater, o.ID, true); err != nil {
			log.Printf("reconcile fulfilled order %s: %v", o.ID, err)
		}
	}
	return nil
}
