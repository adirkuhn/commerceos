package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// StartOrResumePackingUseCase mirrors StartOrResumePickingUseCase -- reuses
// the order's existing in_progress packing session rather than creating a
// duplicate, and enforces the same per-order lock (see SessionLock). Picking
// is optional: an order that skipped the picking module may go straight to
// packing (see SalesOrder.CanStartPacking). An open picking session on the
// order is taken over when its lock allows (stale, the caller's own, or the
// caller is an admin) -- it's closed as abandoned, keeping its scans -- and
// otherwise blocks packing with ErrOrderPickingInProgress.
type StartOrResumePackingUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	sessionRepo    omsDomain.PackingSessionRepository
	pickingRepo    pickingSessionTakeover
	pickScanRepo   pickScanLister
}

// pickingSessionTakeover is what packing needs from PickingSessionRepository
// to find and close the order's open picking session.
type pickingSessionTakeover interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PickingSession, error)
	Update(ctx context.Context, s *omsDomain.PickingSession) error
	ReleaseLock(ctx context.Context, id string) error
}

type pickScanLister interface {
	ListBySession(ctx context.Context, sessionID string) ([]omsDomain.PickScan, error)
}

func NewStartOrResumePackingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, sessionRepo omsDomain.PackingSessionRepository, pickingRepo pickingSessionTakeover, pickScanRepo pickScanLister) *StartOrResumePackingUseCase {
	return &StartOrResumePackingUseCase{db: db, salesOrderRepo: salesOrderRepo, sessionRepo: sessionRepo, pickingRepo: pickingRepo, pickScanRepo: pickScanRepo}
}

func (uc *StartOrResumePackingUseCase) Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) (*omsDomain.PackingSession, error) {
	var session omsDomain.PackingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if err := order.CanStartPacking(); err != nil {
			return err
		}
		now := time.Now()
		if err := uc.takeOverPicking(ctx, salesOrderID, actor, now); err != nil {
			return err
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			if !existing.Lock().CanTakeOver(actor, now) {
				return omsDomain.ErrPackingSessionLocked
			}
			if err := uc.sessionRepo.AcquireLock(ctx, existing.ID, actor.UserID, now); err != nil {
				return err
			}
			existing.ApplyLock(existing.Lock().AcquiredBy(actor.UserID, now))
			session = *existing
			return nil
		}

		session = omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: now,
		}
		session.ApplyLock(omsDomain.SessionLock{}.AcquiredBy(actor.UserID, now))
		return uc.sessionRepo.Create(ctx, &session)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePacking: %w", err)
	}
	return &session, nil
}

// takeOverPicking closes the order's open picking session, if any, as
// abandoned -- refusing while someone else actively holds it (unless actor
// is an admin), or while it still holds unapplied order changes that
// abandoning would drop.
func (uc *StartOrResumePackingUseCase) takeOverPicking(ctx context.Context, salesOrderID string, actor omsDomain.LockActor, now time.Time) error {
	picking, err := uc.pickingRepo.FindActiveBySalesOrder(ctx, salesOrderID)
	if err != nil || picking == nil {
		return err
	}
	if !picking.Lock().CanTakeOver(actor, now) {
		return omsDomain.ErrOrderPickingInProgress
	}
	scans, err := uc.pickScanRepo.ListBySession(ctx, picking.ID)
	if err != nil {
		return err
	}
	if omsDomain.HasPendingPickChanges(scans) {
		return omsDomain.ErrPickingSessionHasPendingChanges
	}
	picking.Status = omsDomain.PickingSessionStatusAbandoned
	picking.CompletedAt = &now
	picking.CompletedBy = actor.UserID
	picking.ApplyLock(omsDomain.SessionLock{})
	if err := uc.pickingRepo.Update(ctx, picking); err != nil {
		return err
	}
	return uc.pickingRepo.ReleaseLock(ctx, picking.ID)
}
