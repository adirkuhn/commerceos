package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type activeSessionFinderForMarkPicked interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PackingSession, error)
}

type activityCreator interface {
	Create(ctx context.Context, a *omsDomain.SalesOrderActivity) error
}

// MarkOrderPickedUseCase is the picking module's list mode: the whole order
// is marked picked in one action -- no scanning, no session. The order then
// leaves the picking list and waits in the packing queue. An open scan-mode
// picking session (e.g. the tenant switched modes) is closed as abandoned,
// under the same takeover rules packing uses. Logged to the order's
// activity feed, in the same transaction.
type MarkOrderPickedUseCase struct {
	outbox       eventrecord.Enqueuer
	db           *gorm.DB
	orderRepo    omsDomain.SalesOrderRepository
	origins      originFinder
	packingRepo  activeSessionFinderForMarkPicked
	packScans    packSessionScanChecker
	pickingRepo  pickingSessionTakeover
	pickScanRepo pickScanLister
	activityRepo activityCreator
}

func NewMarkOrderPickedUseCase(db *gorm.DB, orderRepo omsDomain.SalesOrderRepository, origins originFinder, packingRepo activeSessionFinderForMarkPicked, packScans packSessionScanChecker, pickingRepo pickingSessionTakeover, pickScanRepo pickScanLister, activityRepo activityCreator, outbox eventrecord.Enqueuer) *MarkOrderPickedUseCase {
	return &MarkOrderPickedUseCase{outbox: outbox, db: db, orderRepo: orderRepo, origins: origins, packingRepo: packingRepo, packScans: packScans, pickingRepo: pickingRepo, pickScanRepo: pickScanRepo, activityRepo: activityRepo}
}

func (uc *MarkOrderPickedUseCase) Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.orderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		packing, err := uc.packingRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		now := time.Now()
		inPacking, err := packingBlocksPicking(ctx, uc.packScans, packing, now)
		if err != nil {
			return err
		}
		source, err := sourceOf(ctx, uc.origins, salesOrderID)
		if err != nil {
			return err
		}
		if err := order.CanMarkPicked(source, inPacking); err != nil {
			return err
		}
		if err := abandonOpenPicking(ctx, uc.pickingRepo, uc.pickScanRepo, uc.outbox, salesOrderID, actor, now); err != nil {
			return err
		}
		if err := uc.orderRepo.Update(ctx, salesOrderID, map[string]any{"fulfilled": true, "updated_at": now}); err != nil {
			return err
		}
		if err := recordSalesOrderUpdated(ctx, uc.outbox, salesOrderID,
			eventrecord.NewChanges().Set("fulfilled", order.Fulfilled, true), OrderChangeReasonMarkedPicked); err != nil {
			return err
		}
		activity := omsDomain.SalesOrderActivity{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID, UserID: actor.UserID,
			ActivityType: omsDomain.ActivityTypeMarkedPicked, Summary: "Marked as picked", Metadata: []byte("{}"),
		}
		if err := uc.activityRepo.Create(ctx, &activity); err != nil {
			return err
		}
		return recordActivity(ctx, uc.outbox, activity)
	})
	if err != nil {
		return fmt.Errorf("markOrderPicked: %w", err)
	}
	return nil
}
