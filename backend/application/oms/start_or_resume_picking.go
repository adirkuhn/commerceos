package oms

import (
	"context"
	"fmt"
	"time"

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
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	sessionRepo    omsDomain.PickingSessionRepository
	packingRepo    activePackingSessionFinder
}

// activePackingSessionFinder is the one PackingSessionRepository method
// picking needs -- whether the order is mid-packing.
type activePackingSessionFinder interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PackingSession, error)
}

func NewStartOrResumePickingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, sessionRepo omsDomain.PickingSessionRepository, packingRepo activePackingSessionFinder) *StartOrResumePickingUseCase {
	return &StartOrResumePickingUseCase{db: db, salesOrderRepo: salesOrderRepo, sessionRepo: sessionRepo, packingRepo: packingRepo}
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
		if err := order.CanStartPicking(packing != nil); err != nil {
			return err
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		now := time.Now()
		if existing != nil {
			// Someone else is actively working this session -- refuse
			// unless their lock has gone stale or the caller is an admin
			// (see SessionLock.CanTakeOver), otherwise two people end up
			// scanning into the same order at once.
			if !existing.Lock().CanTakeOver(actor, now) {
				return omsDomain.ErrPickingSessionLocked
			}
			if err := uc.sessionRepo.AcquireLock(ctx, existing.ID, actor.UserID, now); err != nil {
				return err
			}
			existing.ApplyLock(existing.Lock().AcquiredBy(actor.UserID, now))
			if existing.Status == omsDomain.PickingSessionStatusPaused {
				existing.Status = omsDomain.PickingSessionStatusInProgress
				existing.PausedAt = nil
				if err := uc.sessionRepo.Update(ctx, existing); err != nil {
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
		session.ApplyLock(omsDomain.SessionLock{}.AcquiredBy(actor.UserID, now))
		return uc.sessionRepo.Create(ctx, &session)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePicking: %w", err)
	}
	return &session, nil
}
