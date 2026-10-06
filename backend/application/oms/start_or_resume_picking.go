package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// StartOrResumePickingUseCase reuses the order's existing in_progress/paused
// picking session rather than creating a duplicate -- same "pausing one
// order to work another is just start/resume" mechanic as
// procurement.StartOrResumeBookingUseCase. Unlike Booking, there's no
// warehouse-assignment step: SalesOrder.WarehouseID is always set. Picking
// is optional (packing accepts unpicked orders), so it refuses an order
// already packed or mid-packing -- see SalesOrder.CanStartPicking.
type StartOrResumePickingUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	sessionRepo    omsDomain.PickingSessionRepository
	packingRepo    activePackingSessionFinder
	packScans      packSessionScanChecker
}

// activePackingSessionFinder is the one PackingSessionRepository method
// picking needs -- whether the order is mid-packing.
type activePackingSessionFinder interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PackingSession, error)
}

func NewStartOrResumePickingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, sessionRepo omsDomain.PickingSessionRepository, packingRepo activePackingSessionFinder, packScans packSessionScanChecker, outbox eventrecord.Enqueuer) *StartOrResumePickingUseCase {
	return &StartOrResumePickingUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, sessionRepo: sessionRepo, packingRepo: packingRepo, packScans: packScans}
}

func (uc *StartOrResumePickingUseCase) Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) (*omsDomain.PickingSession, error) {
	var session omsDomain.PickingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		packing, err := uc.packingRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		inPacking, err := packingBlocksPicking(ctx, uc.packScans, packing, time.Now())
		if err != nil {
			return err
		}
		if err := order.CanStartPicking(inPacking); err != nil {
			return err
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		now := time.Now()
		// Opening only views -- the lock is taken by the first scan (see
		// claimPickingLock), so browsing orders never locks them.
		if existing != nil {
			if existing.Status == omsDomain.PickingSessionStatusPaused {
				existing.Status = omsDomain.PickingSessionStatusInProgress
				existing.PausedAt = nil
				if err := uc.sessionRepo.Update(ctx, existing); err != nil {
					return err
				}
				if err := recordSession(ctx, uc.outbox, events.PickingSessionResumed, existing.ID, salesOrderID, nil); err != nil {
					return err
				}
			}
			session = *existing
			return nil
		}

		session = omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: now,
		}
		if err := uc.sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		return recordSession(ctx, uc.outbox, events.PickingSessionStarted, session.ID, salesOrderID, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePicking: %w", err)
	}
	return &session, nil
}
