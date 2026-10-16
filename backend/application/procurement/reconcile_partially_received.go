package procurement

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ReconcilePartiallyReceivedInput struct {
	// DryRun reports what would change without writing anything.
	DryRun bool
}

// ReconcilePartiallyReceivedResult lists PO IDs by outcome.
type ReconcilePartiallyReceivedResult struct {
	// Checked counts the partially_received POs looked at.
	Checked int
	// Fixed: every delivery together brought everything -- now received
	// (or would be, on a dry run).
	Fixed []string
	// StillShort: really still missing items; left partially_received.
	StillShort []string
	// BookingInProgress: a delivery is being booked; finishing it settles
	// the status, so it is left alone.
	BookingInProgress []string
}

// ReconcilePartiallyReceivedPurchaseOrdersUseCase repairs POs left
// partially_received by the old booking finish, which judged a PO by the
// finishing booking's scans only: a PO whose later delivery brought the
// rest stayed partially_received. Each PO is judged like
// CompleteBookingUseCase now does -- over the scans of all its completed
// bookings -- in its own transaction. Safe to run again: a received PO is
// not looked at.
type ReconcilePartiallyReceivedPurchaseOrdersUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	bookingRepo  procurementDomain.BookingRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	scanRepo     procurementDomain.BookingScanRepository
	outbox       eventrecord.Enqueuer
}

func NewReconcilePartiallyReceivedPurchaseOrdersUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	bookingRepo procurementDomain.BookingRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	scanRepo procurementDomain.BookingScanRepository,
	outbox eventrecord.Enqueuer,
) *ReconcilePartiallyReceivedPurchaseOrdersUseCase {
	return &ReconcilePartiallyReceivedPurchaseOrdersUseCase{
		db: db, poRepo: poRepo, bookingRepo: bookingRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo, outbox: outbox,
	}
}

type reconcileOutcome int

const (
	reconcileSkipped reconcileOutcome = iota
	reconcileFixed
	reconcileStillShort
	reconcileBookingInProgress
)

func (uc *ReconcilePartiallyReceivedPurchaseOrdersUseCase) Execute(ctx context.Context, in ReconcilePartiallyReceivedInput) (*ReconcilePartiallyReceivedResult, error) {
	var ids []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		pos, err := uc.poRepo.List(ctx)
		if err != nil {
			return err
		}
		for _, po := range pos {
			if po.Status == procurementDomain.PurchaseOrderStatusPartiallyReceived {
				ids = append(ids, po.ID)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reconcilePartiallyReceived: %w", err)
	}

	result := &ReconcilePartiallyReceivedResult{Checked: len(ids)}
	for _, id := range ids {
		var outcome reconcileOutcome
		err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			var err error
			outcome, err = uc.reconcile(ctx, id, in.DryRun)
			return err
		})
		if err != nil {
			return result, fmt.Errorf("reconcilePartiallyReceived: purchase order %s: %w", id, err)
		}
		switch outcome {
		case reconcileFixed:
			result.Fixed = append(result.Fixed, id)
		case reconcileStillShort:
			result.StillShort = append(result.StillShort, id)
		case reconcileBookingInProgress:
			result.BookingInProgress = append(result.BookingInProgress, id)
		}
	}
	return result, nil
}

// reconcile judges one PO; call inside tenant.WithTenant.
func (uc *ReconcilePartiallyReceivedPurchaseOrdersUseCase) reconcile(ctx context.Context, id string, dryRun bool) (reconcileOutcome, error) {
	po, err := uc.poRepo.FindByID(ctx, id)
	if err != nil {
		return reconcileSkipped, err
	}
	// Re-checked in this transaction: it may have changed since the list.
	if po.Status != procurementDomain.PurchaseOrderStatusPartiallyReceived {
		return reconcileSkipped, nil
	}
	active, err := uc.bookingRepo.FindActiveByPurchaseOrder(ctx, id)
	if err != nil {
		return reconcileSkipped, err
	}
	if active != nil {
		return reconcileBookingInProgress, nil
	}
	bookings, err := uc.bookingRepo.ListCompletedByPurchaseOrder(ctx, id)
	if err != nil {
		return reconcileSkipped, err
	}
	if len(bookings) == 0 {
		return reconcileStillShort, nil
	}
	var scans []procurementDomain.BookingScan
	for _, b := range bookings {
		bookingScans, err := uc.scanRepo.ListByBooking(ctx, b.ID)
		if err != nil {
			return reconcileSkipped, err
		}
		scans = append(scans, bookingScans...)
	}
	lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, id)
	if err != nil {
		return reconcileSkipped, err
	}
	if len(procurementDomain.Shortfalls(lineItems, scans)) > 0 {
		return reconcileStillShort, nil
	}
	if dryRun {
		return reconcileFixed, nil
	}

	before := *po
	po.Status = procurementDomain.PurchaseOrderStatusReceived
	if err := uc.poRepo.Update(ctx, po); err != nil {
		return reconcileSkipped, err
	}
	if err := eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderUpdated, po.ID, purchaseOrderChanges(before, *po),
		eventrecord.Data{"reason": PurchaseOrderChangeReasonDeliveriesReconciled}); err != nil {
		return reconcileSkipped, err
	}
	return reconcileFixed, nil
}
